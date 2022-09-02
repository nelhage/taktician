#!/usr/bin/env python
import argparse
import time
import typing as T  # noqa
import os.path
import yaml

import torch

import grpc
from tak.proto import analysis_pb2_grpc
import asyncio

import xformer
from xformer import data, model, train, loading
from xformer.data import Dataset
from xformer.train import hooks, lr_schedules
from tak.model import batches, heads, losses

import tak.model.server
from tak import self_play, mcts
from tak.alphazero import trainer, hooks
from tak import alphazero
from xformer import yaml_ext  # noqa


def parse_args():
    parser = argparse.ArgumentParser(description="Train a Tak player using self-play")

    parser.add_argument("--layers", type=int, default=2, help="Number of layers")
    parser.add_argument("--d_model", type=int, default=None, help="embedding dimension")
    parser.add_argument("--d_head", type=int, default=32, help="head dimension")
    parser.add_argument(
        "--n_ctx", type=int, default=1024, help="maximum context length"
    )
    parser.add_argument(
        "--pe", type=str, default=None, help="positional encoding (sin, learned, none)"
    )

    parser.add_argument("--batch", type=int, default=64, help="batch size")

    parser.add_argument(
        "--device", type=str, choices=("cpu", "cuda"), default="cuda", help="device"
    )

    parser.add_argument("--lr", type=float, default=5e-4, help="learning rate")

    parser.add_argument("--steps", type=int, default=10)

    parser.add_argument("--size", type=int, default=3)

    parser.add_argument("--rollouts-per-step", type=int, default=1000)
    parser.add_argument("--replay-buffer-steps", type=int, default=4)
    parser.add_argument("--train-positions", type=int, default=1024)

    parser.add_argument("--rollout-workers", type=int, default=50)
    parser.add_argument("--rollout-simulations", type=int, default=25)

    parser.add_argument("--run-dir", type=str, metavar="PATH")
    parser.add_argument("--save-freq", type=int, metavar="STEPS", default=10)

    parser.add_argument("--test-data", type=str, metavar="PATH")
    parser.add_argument("--test-freq", type=int, metavar="STEPS", default=10)

    parser.add_argument("--progress", default=True, action="store_true")
    parser.add_argument("--no-progress", dest="progress", action="store_false")

    parser.add_argument("--job-name", type=str, default=None, help="job name for wandb")
    parser.add_argument(
        "--project", type=str, default="taktician-alphazero", help="project for wandb"
    )
    parser.add_argument("--wandb", action="store_true", default=False)
    parser.add_argument("--no-wandb", action="store_false", dest="wandb")
    parser.add_argument("--load-model", type=str, help="Initial model to load")

    return parser.parse_args()


def main():
    args = parse_args()

    if args.run_dir and os.path.exists(os.path.join(args.run_dir, "run.yaml")):
        with open(os.path.join(args.run_dir, "run.yaml"), "r") as fh:
            config = yaml.unsafe_load(fh)

    else:
        model_cfg = xformer.Config(
            n_layer=args.layers,
            d_model=args.d_model or 128 * args.layers,
            d_head=args.d_head,
            n_ctx=args.n_ctx,
            n_vocab=256,
            autoregressive_mask=False,
            output_head=heads.PolicyValue,
        )
        if args.pe is not None:
            model_cfg.positional_encoding = args.pe

        run_hooks = alphazero.config.default_hooks()
        if args.wandb:
            run_hooks.append(
                hooks.WandB(
                    job_name=args.job_name,
                    project=args.project,
                )
            )
        run_hooks.append(hooks.SavingHook(freq=args.save_freq))
        if args.test_data:
            run_hooks.append(
                hooks.TestLoss(
                    dataset=Dataset(
                        path=os.path.realpath(args.test_data),
                        batch_size=args.batch,
                        device=args.device,
                        batch_class=batches.PositionValuePolicy,
                    ),
                    frequency=args.test_freq,
                )
            )

        config = alphazero.Config(
            model=model_cfg,
            device=args.device,
            load_model=args.load_model,
            run_dir=args.run_dir,
            size=args.size,
            rollout_workers=args.rollout_workers,
            rollouts_per_step=args.rollouts_per_step,
            rollout_resignation_threshold=0.99,
            rollout_ply_limit=20,
            replay_buffer_steps=args.replay_buffer_steps,
            train_batch=args.batch,
            train_positions=args.train_positions,
            lr=args.lr,
            train_steps=args.steps,
            hooks=run_hooks,
        )
        config.rollout_config.simulation_limit = args.rollout_simulations
        config.rollout_config.time_limit = 0

    train = trainer.TrainingRun(config=config)
    train.run()


if __name__ == "__main__":
    main()
