import argparse
import sys

import tak
from tak import mcts
from xformer import loading
from tak.model import wrapper, grpc
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
        "--model",
        type=str,
    )
    parser.add_argument(
        "--host",
        type=str,
    )
    parser.add_argument(
        "--port",
        type=int,
        default=5001,
    )

    args = parser.parse_args(argv)

    if (args.model and args.host) or not (args.model or args.host):
        raise ValueError("Must specify either --host or --model, not both")
    if args.model:
        model = loading.load_model(args.model, args.device)
        if args.fp16:
            model = model.to(torch.float16)

        if args.graph:
            network = wrapper.GraphedWrapper(model)
        else:
            network = wrapper.ModelWrapper(model, device=args.device)
    else:
        network = grpc.GRPCNetwork(host=args.host, port=args.port)

    p = tak.Position.from_config(tak.Config(size=args.size))

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
