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


@define
class Transcript:
    positions: list[tak.Position] = field(factory=list)
    moves: list[list[tak.Move]] = field(factory=list)
    probs: list[torch.Tensor] = field(factory=list)


def self_play(engine, size=3):
    p = tak.Position.from_config(tak.Config(size=size))

    log = Transcript()

    tree = mcts.Node(position=p, move=None)

    while True:
        color, over = tree.position.winner()
        if over is not None:
            break

        tree = engine.analyze_tree(tree)
        probs = engine.tree_probs(tree)

        log.positions.append(tree.position)
        log.moves.append([c.move for c in tree.children])
        log.probs.append(probs)

        tree = tree.children[torch.multinomial(probs, 1).item()]

    return log


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
        "--threads",
        type=int,
        default=1,
    )

    args = parser.parse_args(argv)

    network = grpc.GRPCNetwork(host=args.host, port=args.port)

    engine = mcts.MCTS(
        mcts.Config(
            network=network,
            simulation_limit=args.simulations,
            time_limit=0,
        )
    )

    start = time.time()
    with concurrent.futures.ThreadPoolExecutor(args.threads) as tpe:
        futs = [
            tpe.submit(partial(self_play, engine, args.size)) for _ in range(args.games)
        ]
        logs = [f.result() for f in futs]
    end = time.time()

    print(
        f"done games={len(logs)} plies={sum(len(l.positions) for l in logs)} threads={args.threads} duration={end-start:.2f}"
    )


if __name__ == "__main__":
    main(sys.argv[1:])
