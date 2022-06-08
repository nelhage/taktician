import xformer
import xformer.data

import os
import torch
import time
import itertools
import argparse
import wandb
from contextlib import nullcontext

from torch.profiler import profile, ProfilerAction


class Dataset:
    def __init__(self, path):
        self.data = torch.load(path)

    def __len__(self):
        return len(next(iter(self.data.values())))

    def __getitem__(self, i):
        return {k: v[i] for (k, v) in self.data.items()}


def fwd_and_loss(model, xent, record):
    batch = record["positions"].to(device=model.device, dtype=torch.long)
    logits = model(batch[:, :-1])
    targets = batch[:, 1:]
    loss = (
        xent(logits.permute(0, 2, 1), targets)
        * record["mask"].to(device=model.device)[:, :-1]
    ).mean()
    return loss


def main():
    parser = argparse.ArgumentParser(description="Train a transformer")
    parser.add_argument("--layers", type=int, default=2, help="Number of layers")
    parser.add_argument("--d_model", type=int, default=None, help="embedding dimension")
    parser.add_argument("--d_head", type=int, default=32, help="head dimension")
    parser.add_argument(
        "--n_ctx", type=int, default=1024, help="maximum context length"
    )
    parser.add_argument(
        "--pe", type=str, default=None, help="positional encoding (sin, learned, none)"
    )

    parser.add_argument("--data", type=str, default="data/positions", help="datasource")

    parser.add_argument("--batch", type=int, default=64, help="batch size")
    parser.add_argument("--minibatch", type=int, default=4, help="minibatch")

    parser.add_argument(
        "--test-batches", type=int, default=16, help="number of test minibatches"
    )
    parser.add_argument(
        "--test-freq", type=int, default=1000, help="measure test loss every N steps"
    )

    parser.add_argument(
        "--device", type=str, choices=("cpu", "cuda"), default="cuda", help="device"
    )

    parser.add_argument("--wandb", action="store_true", default=False)
    parser.add_argument("--no-wandb", action="store_false", dest="wandb")

    parser.add_argument("--lr", type=float, default=0.001, help="learning rate")
    parser.add_argument("--steps", type=int, default=None)
    parser.add_argument("--profile-steps", type=str, default=None)
    parser.add_argument("--positions", type=int, default=None)

    args = parser.parse_args()

    cfg = xformer.Config(
        n_layer=args.layers,
        d_model=args.d_model or 128 * args.layers,
        d_head=args.d_head,
        n_ctx=args.n_ctx,
        n_vocab=256,
        autoregressive_mask=True,
    )
    if args.pe is not None:
        cfg.positional_encoding = args.pe

    train_ds = Dataset(args.data + "-train.pt")
    loader = torch.utils.data.DataLoader(
        train_ds,
        batch_size=args.minibatch,
        pin_memory=True,
        num_workers=1,
        shuffle=True,
    )

    test_ds = Dataset(args.data + "-test.pt")
    test_loader = torch.utils.data.DataLoader(
        test_ds,
        batch_size=args.minibatch,
        pin_memory=True,
        num_workers=1,
    )
    test_batches = list(itertools.islice(test_loader, args.test_batches))

    model = xformer.Transformer(cfg, dtype=torch.float32, device=args.device)

    xent = torch.nn.CrossEntropyLoss(reduction="none")
    opt = torch.optim.AdamW(model.parameters(), lr=args.lr)

    assert args.batch % args.minibatch == 0, "minibatch must divide batch"
    steps_per_batch = args.batch // args.minibatch

    data = iter(loader)

    if args.wandb:
        run = wandb.init()  # noqa
        wandb.watch(model, log_freq=100, log="gradients")
        wandb.config.update(args)
        wandb.config.update({"n_parameters": cfg.n_parameters})

    model.init_weights()
    param_bytes = 0
    for p in model.parameters():
        param_bytes += p.numel() * p.element_size()

    print(
        f"Training a {cfg.n_layer}L model with {cfg.n_parameters:,} non-embedding parameters..."
    )
    print(f" Model params use {param_bytes/1024**3:.2f}GiB on device")

    start = time.time()
    positions = 0

    steps = range(args.steps) if args.steps is not None else itertools.count()

    profile_steps = set()
    if args.profile_steps is not None:
        profile_steps = set(int(s) for s in args.profile_steps.split(","))

    def schedule(step):
        if step in profile_steps:
            print(f"Profiling step {step}...")
            return ProfilerAction.RECORD_AND_SAVE
        if step + 1 in profile_steps:
            return ProfilerAction.WARMUP
        return ProfilerAction.NONE

    def save_profile(prof):
        os.makedirs("profile", 0o755, True)
        prof.export_chrome_trace(f"profile/step_{step_i}.pt.trace.json")

    profiler = profile(schedule=schedule, with_stack=True, on_trace_ready=save_profile)

    with profiler:
        for step_i in steps:
            step_start = time.time()

            avg_loss = torch.zeros((), device=args.device)
            opt.zero_grad(set_to_none=True)
            for _ in range(steps_per_batch):
                batch = next(data)
                loss = fwd_and_loss(model, xent, batch)
                avg_loss += loss
                positions += batch["positions"].size(0)
                (loss / steps_per_batch).backward()
            opt.step()
            profiler.step()

            now = time.time()
            avg_loss = (avg_loss / steps_per_batch).item()
            print(
                f"[step={step_i:06d} t={now-start:.1f}s positions={positions:08d}] loss={avg_loss:2.2f} ms_per_step={1000*(now-step_start):.0f}"
            )
            if args.wandb:
                wandb.log(
                    {
                        "positions": positions,
                        "elapsed_time": now - start,
                        "train_loss": avg_loss,
                    },
                    step=step_i,
                )
            if args.positions is not None and positions >= args.positions:
                break

            if step_i % args.test_freq == 0:
                test_loss = torch.tensor(
                    [fwd_and_loss(model, xent, b).item() for b in test_batches]
                ).mean()
                print(f"[step={step_i:06d}] test_loss={test_loss:4.2f}")


if __name__ == "__main__":
    main()
