#!/usr/bin/env python
import argparse
import sys
import traceback

import tak
from tak import mcts, self_play
from tak.model import grpc
import attrs
import tqdm

import queue
from torch import multiprocessing

import torch
import numpy as np

import time


def main(argv):
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--simulations",
        dest="simulations",
        type=int,
        default=100,
        metavar="POSITIONS",
    )
    parser.add_argument(
        "--size",
        dest="size",
        type=int,
        default=3,
        metavar="SIZE",
    )
    parser.add_argument(
        "--host",
        type=str,
        default="localhost",
    )
    parser.add_argument(
        "--port",
        type=int,
        default=5001,
    )
    parser.add_argument(
        "--games",
        type=int,
        default=1,
    )
    parser.add_argument(
        "--noise-alpha",
        type=float,
        default=None,
    )
    parser.add_argument(
        "--noise-weight",
        type=float,
        default=0.25,
    )
    parser.add_argument(
        "--resign-threshold",
        type=float,
        default=0.99,
    )
    parser.add_argument(
        "--threads",
        type=int,
        default=1,
    )
    parser.add_argument(
        "-C",
        type=float,
        default=4,
    )
    parser.add_argument("--write-games", type=str, metavar="FILE")

    args = parser.parse_args(argv)

    config = self_play.SelfPlayConfig(
        size=args.size,
        workers=args.threads,
        resignation_threshold=args.resign_threshold,
        engine_factory=self_play.BuildRemoteMCTS(
            host=args.host,
            port=args.port,
            config=mcts.Config(
                simulation_limit=args.simulations,
                root_noise_alpha=args.noise_alpha,
                root_noise_mix=args.noise_weight,
                C=args.C,
            ),
        ),
    )

    start = time.time()

    logs = self_play.play_many_games(config, args.games, progress=True)

    end = time.time()

    stats = mcts.Stats()
    for l in logs:
        stats = stats.merge(l.stats)

    print(
        f"done games={len(logs)}"
        f" plies={sum(len(l.positions) for l in logs)}"
        f" threads={args.threads} duration={end-start:.2f}"
        f" games/s={args.games/(end-start):.1f}"
        " " + " ".join(f"{k}={v}" for (k, v) in attrs.asdict(stats).items())
    )

    if args.write_games:
        torch.save(
            self_play.encode_games(logs),
            args.write_games,
        )

        pass


if __name__ == "__main__":
    main(sys.argv[1:])
