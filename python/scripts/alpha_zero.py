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
from xformer.train import hooks, lr_schedules
from tak.model import batches, heads, losses

import tak.model.server
from tak import self_play, mcts
from tak.alphazero import trainer
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

    if args.run_dir and os.path.exists(os.path.join(args.run_dir, "latest")):
        print("Run directory exists, resuming...")
        model_cfg = loading.load_config(os.path.join(args.run_dir, "latest"))
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

        config = alphazero.Config(
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
            save_freq=args.save_freq,
            train_steps=args.steps,
            wandb=args.wandb,
            project=args.project,
            job_name=args.job_name,
        )
        config.rollout_config.simulation_limit = args.rollout_simulations
        config.rollout_config.time_limit = 0

    model = xformer.Transformer(model_cfg, device=config.device)

    train = trainer.TrainingRun(config=config, model=model)
    train.run()


if __name__ == "__main__":
    main()
