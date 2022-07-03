import argparse
import time
import typing as T  # noqa

import torch
from torch import multiprocessing

import grpc
from tak.proto import analysis_pb2_grpc
import asyncio

import xformer
from xformer import data, model, train, loading
from xformer.train import hooks, lr_schedules
from tak.model import batches, heads, losses

import tak.model.server
from tak import self_play
from tak.alphazero import model_process
from tak import alphazero


def parse_args():
    parser = argparse.ArgumentParser(description="Train a Tak player using self-play")

    parser.add_argument("--load-model", type=str, help="Initial model to load")
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

    parser.add_argument("--job-name", type=str, default=None, help="job name for wandb")
    parser.add_argument("--group", type=str, default=None, help="wandb group name")
    parser.add_argument("--wandb", action="store_true", default=False)
    parser.add_argument("--no-wandb", action="store_false", dest="wandb")

    parser.add_argument("--lr", type=float, default=5e-4, help="learning rate")

    parser.add_argument("--rollouts-per-step", type=int, default=1000)
    parser.add_argument("--replay-buffer-steps", type=int, default=4)
    parser.add_argument("--train-positions", type=int, default=1024)

    parser.add_argument("--size", type=int, default=3)

    parser.add_argument("--rollout-workers", type=int, default=50)
    parser.add_argument("--rollout-simulations", type=int, default=25)

    return parser.parse_args()


def main():
    multiprocessing.set_start_method("spawn")

    args = parse_args()

    if args.load_model:
        train_model = loading.load_model(args.load_model, device=args.device)
    else:
        cfg = xformer.Config(
            n_layer=args.layers,
            d_model=args.d_model or 128 * args.layers,
            d_head=args.d_head,
            n_ctx=args.n_ctx,
            n_vocab=256,
            autoregressive_mask=False,
            output_head=heads.PolicyValue,
        )
        if args.pe is not None:
            cfg.positional_encoding = args.pe
        train_model = model.Transformer(cfg, dtype=torch.float32, device=args.device)
        train_model.init_weights()

    config = alphazero.Config(
        device=args.device,
        server_port=5001,
        size=args.size,
        rollout_workers=args.rollout_workers,
        rollout_simulations=args.rollout_simulations,
        rollouts_per_step=args.rollouts_per_step,
        replay_buffer_steps=args.replay_buffer_steps,
        train_batch=args.batch,
        train_positions=args.train_positions,
    )

    del args

    # run self-play to get a batch of games
    srv = model_process.create_server(model=train_model, config=config)
    srv.start()

    rollout_config = self_play.SelfPlayConfig(
        size=config.size,
        games=config.rollouts_per_step,
        workers=config.rollout_workers,
        engine_factory=self_play.BuildRemoteMCTS(
            host="localhost",
            port=config.server_port,
            simulations=config.rollout_simulations,
        ),
    )

    start = time.time()

    logs = self_play.play_many_games(rollout_config, progress=True)
    plies = sum(len(l.positions) for l in logs)
    end = time.time()

    print(
        f"generated games={config.rollouts_per_step}"
        f" plies={plies}"
        f" in {end-start:0.2f}s"
        f" ply/s={plies/(end-start):.1f}s"
    )

    batch = self_play.encode_games(logs)
    batch["positions"] = batch["positions"].to(torch.long)
    srv.train_step({k: v.share_memory_() for (k, v) in batch.items()})

    srv.stop()


if __name__ == "__main__":
    main()
