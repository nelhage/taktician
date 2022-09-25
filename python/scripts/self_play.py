#!/usr/bin/env python
import argparse
import sys
import traceback

import tak
from tak import mcts, self_play
from tak.model import grpc
from attrs import define, field
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
        "--threads",
        type=int,
        default=1,
    )
    parser.add_argument("--write-games", type=str, metavar="FILE")

    args = parser.parse_args(argv)

    config = self_play.SelfPlayConfig(
        size=args.size,
        workers=args.threads,
        engine_factory=self_play.BuildRemoteMCTS(
            host=args.host,
            port=args.port,
            config=mcts.Config(
                simulation_limit=args.simulations,
                root_noise_alpha=args.noise_alpha,
                root_noise_mix=args.noise_weight,
            ),
        ),
    )

    start = time.time()

    logs = self_play.play_many_games(config, args.games, progress=True)

    end = time.time()

    print(
        f"done games={len(logs)} plies={sum(len(l.positions) for l in logs)} threads={args.threads} duration={end-start:.2f} games/s={args.games/(end-start):.1f}"
    )

    if args.write_games:
        torch.save(
            self_play.encode_games(logs),
            args.write_games,
        )

        pass


if __name__ == "__main__":
    main(sys.argv[1:])
