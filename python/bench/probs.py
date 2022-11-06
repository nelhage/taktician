import argparse
import sys
from functools import partial

import tak
from tak import mcts
from tak.model import grpc
from attrs import define, field

import concurrent.futures

import torch

import time


def main(argv):
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--iterations",
        dest="iterations",
        type=int,
        default=100,
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

    args = parser.parse_args(argv)

    network = grpc.GRPCNetwork(host=args.host, port=args.port)

    engine = mcts.MCTS(
        mcts.Config(
            network=network,
            simulation_limit=1,
            time_limit=0,
        )
    )

    tree = engine.analyze(tak.Position.from_config(tak.Config(size=args.size)))

    start = time.perf_counter()
    for _ in range(args.iterations):
        engine.tree_probs(tree)

    end = time.perf_counter()

    print(
        f"done loops={args.iterations}"
        + f" duration={end-start:.2f}"
        + f" us/loop={1_000_000*(end-start)/args.iterations:.2f}"
    )


if __name__ == "__main__":
    main(sys.argv[1:])
