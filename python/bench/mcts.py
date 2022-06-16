import argparse
import sys

import tak
from tak import mcts
from xformer import loading
from tak.model import wrapper
import torch

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
        "--graph",
        action="store_true",
        default=False,
        help="Use CUDA graphs to run the network",
    )
    parser.add_argument(
        "--fp16",
        action="store_true",
        default=False,
        help="Run model in float16",
    )
    parser.add_argument(
        "--device",
        type=str,
        default="cpu",
    )
    parser.add_argument(
        "model",
        type=str,
    )

    args = parser.parse_args(argv)

    model = loading.load_model(args.model, args.device)
    if args.fp16:
        model = model.to(torch.float16)

    p = tak.Position.from_config(tak.Config(size=args.size))

    if args.graph:
        network = wrapper.GraphedWrapper(model)
    else:
        network = wrapper.ModelWrapper(model, device=args.device)

    engine = mcts.MCTS(
        mcts.Config(
            network=network,
            simulation_limit=args.simulations,
            time_limit=0,
        )
    )

    start = time.time()
    tree = engine.analyze(p)
    end = time.time()

    print(f"done simulations={tree.simulations} duration={end-start:.2f}")


if __name__ == "__main__":
    main(sys.argv[1:])
