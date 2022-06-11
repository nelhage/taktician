import xformer

from tak.model import encoding

from xformer import data, train, model
from xformer.train import hooks

from attrs import define

import torch
from torch import nn
from torch.nn import functional as F
import argparse

import typing as T  # noqa


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

    def train_and_metrics(self, batch, logits):
        return (
            (self.xent(logits.permute(0, 2, 1), batch.targets) * batch.mask).mean(),
            {},
        )


OUTPUT_SENTINEL = 256


@define
class PositionValuePolicyBatch:
    data: dict[str, torch.Tensor]

    @property
    def inputs(self):
        return F.pad(self.data["positions"], (0, 1), value=OUTPUT_SENTINEL)

    @property
    def moves(self):
        return self.data["moves"]

    @property
    def moves_mask(self):
        return self.data["moves_mask"]

    @property
    def values(self):
        return self.data["values"]


class PolicyValueLoss:
    def __init__(self):
        self.xent = torch.nn.CrossEntropyLoss(reduction="none")

    def loss_and_metrics(self, batch, logits):
        v_logits = logits["values"]
        m_logits = logits["moves"]

        # breakpoint()

        with torch.no_grad():
            argmax = torch.max(m_logits, dim=-1).indices
            match = torch.where(
                batch.moves_mask != 0,
                (argmax == batch.moves),
                torch.ones_like(argmax, dtype=torch.bool),
            )
            all_match = torch.prod(match, dim=-1).float().mean()

        v_error = F.mse_loss(v_logits, batch.values)

        return (
            v_error
            + (
                self.xent(m_logits.permute(0, 2, 1), batch.moves) * batch.moves_mask
            ).mean()
        ), {
            "v_error": v_error.item(),
            "acc@1": all_match.mean().item(),
        }


class PolicyValueHead(nn.Module):
    def __init__(self, cfg, dtype=None, device=None):
        super().__init__()
        self.final_ln = nn.LayerNorm(
            normalized_shape=(cfg.d_model,), dtype=dtype, device=device
        )
        self.v_proj = nn.Linear(cfg.d_model, 1, dtype=dtype, device=device)
        self.move_proj = nn.Linear(
            cfg.d_model, 3 * encoding.MAX_SLIDES, dtype=dtype, device=device
        )

    def init_weights(self, cfg):
        pass

    def forward(self, acts):
        acts = self.final_ln(acts)[:, -1]

        v = torch.tanh(self.v_proj(acts))

        moves = self.move_proj(acts).reshape(-1, 3, encoding.MAX_SLIDES)

        return {
            "values": v.squeeze(-1),
            "moves": moves,
        }


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
        output_head=PolicyValueHead,
    )
    if args.pe is not None:
        cfg.positional_encoding = args.pe

    train_ds = data.Dataset(
        args.data + "-train.pt",
        batch_size=args.batch,
        device=args.device,
        batch_class=PositionValuePolicyBatch,
    )
    test_ds = data.Dataset(
        args.data + "-test.pt",
        batch_size=args.batch,
        device=args.device,
        batches=args.test_batches,
        batch_class=PositionValuePolicyBatch,
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
        loss=PolicyValueLoss(),
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
