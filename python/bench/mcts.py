import argparse
import sys

import tak
from tak import mcts
from xformer import loading
from tak.model import wrapper


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

    p = tak.Position.from_config(tak.Config(size=args.size))

    engine = mcts.MCTS(
        mcts.Config(
            network=wrapper.ModelWrapper(model, device=args.device),
            simulation_limit=args.simulations,
            time_limit=0,
        )
    )

    engine.get_move(p)


if __name__ == "__main__":
    main(sys.argv[1:])
