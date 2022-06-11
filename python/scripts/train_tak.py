import xformer

from tak.model import heads, batches, losses

from xformer import data, train, model
from xformer.train import hooks

import attrs
from attrs import define

import torch
import argparse
import yaml
import json
import os.path

import typing as T  # noqa


@define
class LRSchedule:
    warmup_steps: int
    cooldown_steps: int
    cooldown_start: int

    def __call__(self, stats):
        if stats.step < self.warmup_steps:
            return stats.step / self.warmup_steps
        if stats.step > self.cooldown_start:
            end = self.cooldown_start + self.cooldown_steps
            remaining = end - stats.step
            return (remaining + 1) / self.cooldown_steps
        return 1.0


@define
class SaveHook(train.Hook):
    save_dir: str
    step_freq: int

    def after_step(self, run, stats):
        if stats.step % self.step_freq != 0:
            return
        self.save_run(run, stats)

    def after_run(self, run, stats):
        self.save_run(run, stats)

    def save_run(self, run: train.Run, stats: train.Stats):
        run_dir = os.path.join(self.save_dir, f"step_{stats.step:06d}")
        os.makedirs(run_dir, exist_ok=True)
        print(f"Saving to {run_dir}...")
        torch.save(
            run.model.state_dict(),
            os.path.join(run_dir, "model.pt"),
        )
        with open(os.path.join(run_dir, "config.yaml"), "w") as fh:
            yaml.dump(run.model.cfg, fh)
        with open(os.path.join(run_dir, "stats.json"), "w") as fh:
            json.dump(attrs.asdict(stats), fh, indent=2)
        # torch.save(os.path.join(run_dir, "model.opt.pt"), run.optimizer.state_dict())


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
        "--test-freq", type=int, default=100, help="measure test loss every N steps"
    )

    parser.add_argument(
        "--save-freq", type=int, default=100, help="save model every N steps"
    )
    parser.add_argument("--save-dir", type=str, help="save directory")

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
        n_vocab=257,
        autoregressive_mask=False,
        output_head=heads.PolicyValue,
    )
    if args.pe is not None:
        cfg.positional_encoding = args.pe

    train_ds = data.Dataset(
        args.data + "-train.pt",
        batch_size=args.batch,
        device=args.device,
        batch_class=batches.PositionValuePolicy,
    )
    test_ds = data.Dataset(
        args.data + "-test.pt",
        batch_size=args.batch,
        device=args.device,
        batches=args.test_batches,
        batch_class=batches.PositionValuePolicy,
    )

    extra_hooks = []
    if args.wandb:
        extra_hooks.append(
            hooks.Wandb(
                project="taktician",
                job_name=args.job_name,
                group=args.group,
                config=args,
            )
        )
    if args.profile_steps:
        extra_hooks.append(
            hooks.Profile(
                extra_steps=set(map(int, args.profile_steps.split(","))),
            )
        )
    if args.save_dir:
        extra_hooks.append(
            SaveHook(
                save_dir=args.save_dir,
                step_freq=args.save_freq,
            )
        )

    if args.steps:
        warmup_frac = 0.05
        cooldown_frac = 0.8
        schedule = LRSchedule(
            warmup_steps=int(warmup_frac * args.steps),
            cooldown_start=int((1 - cooldown_frac) * args.steps),
            cooldown_steps=int(cooldown_frac * args.steps),
        )
    else:
        schedule = None

    run = train.Run(
        model=model.Transformer(cfg, dtype=torch.float32, device=args.device),
        dataset=train_ds,
        # loss=MaskedARLoss(),
        loss=losses.PolicyValue(),
        optimizer=train.Optimizer(lr=args.lr, lr_schedule=schedule),
        stop=train.StopTrigger(steps=args.steps, sequences=args.positions),
        hooks=[
            hooks.TestLoss(test_ds, args.test_freq),
        ]
        + extra_hooks,
    )

    print(
        f"Training a {cfg.n_layer}L model with {cfg.n_parameters:,} non-embedding parameters..."
    )
    param_bytes = sum(t.numel() * t.element_size() for t in run.model.parameters())
    print(f" Model params use {param_bytes/1024**3:.2f}GiB on device")

    trainer = train.Trainer(run)
    trainer.train()


if __name__ == "__main__":
    main()
