import xformer
import xformer.train.wandb

from xformer import data, train

from attrs import define

import os
import torch
import argparse

import typing as T

from torch.profiler import profile, ProfilerAction


@define
class PositionBatch:
    data: dict[str, torch.Tensor]

    @property
    def inputs(self):
        return self.data["positions"][:, :-1]

    @property
    def targets(self):
        return self.data["positions"][:, 1:]

    @property
    def mask(self):
        return self.data["mask"][:, :-1]


class MaskedARLoss:
    def __init__(self):
        self.xent = torch.nn.CrossEntropyLoss(reduction="none")

    def __call__(self, batch, logits):
        return (self.xent(logits.permute(0, 2, 1), batch.targets) * batch.mask).mean()


@define
class StopTrigger:
    steps: T.Optional[int]
    sequences: T.Optional[int]

    def __call__(self, stats: train.Stats):
        if self.steps is not None and stats.step >= self.steps:
            return True
        if self.sequences is not None and stats.sequences >= self.sequences:
            return True
        return False


class TestLossHook(train.Hook):
    def __init__(self, dataset, freq: int):
        self.dataset = dataset
        self.frequency = freq

    def after_step(self, run: train.Run, stats: train.Stats):
        if stats.step % self.frequency != 0:
            return

        test_loss = (
            torch.tensor(
                [
                    run.loss(batch, run.model(batch.inputs)).item()
                    for batch in self.dataset
                ]
            )
            .mean()
            .item()
        )
        print(f"[step={stats.step:06d}] test_loss={test_loss:4.2f}")
        stats.metrics["test_loss"] = test_loss


def parse_args():
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

    parser.add_argument("--batch", type=int, default=4, help="batch size")

    parser.add_argument(
        "--test-batches", type=int, default=16, help="number of test batches"
    )
    parser.add_argument(
        "--test-freq", type=int, default=1000, help="measure test loss every N steps"
    )

    parser.add_argument(
        "--device", type=str, choices=("cpu", "cuda"), default="cuda", help="device"
    )

    parser.add_argument("--job-name", type=str, default=None, help="job name for wandb")
    parser.add_argument("--group", type=str, default=None, help="wandb group name")
    parser.add_argument("--wandb", action="store_true", default=False)
    parser.add_argument("--no-wandb", action="store_false", dest="wandb")

    parser.add_argument("--lr", type=float, default=5e-4, help="learning rate")
    parser.add_argument("--steps", type=int, default=None)
    parser.add_argument("--profile-steps", type=str, default=None)
    parser.add_argument("--positions", type=int, default=None)

    return parser.parse_args()


def main():
    args = parse_args()

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

    train_ds = data.Dataset(
        args.data + "-train.pt",
        batch_size=args.batch,
        device=args.device,
        batch_class=PositionBatch,
    )
    test_ds = data.Dataset(
        args.data + "-test.pt",
        batch_size=args.batch,
        device=args.device,
        batches=args.test_batches,
        batch_class=PositionBatch,
    )

    model = xformer.Transformer(cfg, dtype=torch.float32, device=args.device)

    extra_hooks = []
    if args.wandb:
        extra_hooks.append(
            train.wandb.WandbHook(
                project="taktician",
                job_name=args.job_name,
                group=args.group,
                config=args,
            )
        )

    run = train.Run(
        model=model,
        dataset=train_ds,
        loss=MaskedARLoss(),
        optimizer=train.Optimizer(lr=args.lr),
        stop=StopTrigger(steps=args.steps, sequences=args.positions),
        hooks=[
            TestLossHook(test_ds, args.test_freq),
        ]
        + extra_hooks,
    )

    print(
        f"Training a {cfg.n_layer}L model with {cfg.n_parameters:,} non-embedding parameters..."
    )
    param_bytes = sum(t.numel() * t.element_size() for t in model.parameters())
    print(f" Model params use {param_bytes/1024**3:.2f}GiB on device")

    trainer = train.Trainer(run)
    trainer.train()


if __name__ == "__main__":
    main()


def dumping_ground():
    ##########

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
