import tak
from tak import mcts
import optparse

import sys


def main(args):
    parser = optparse.OptionParser()
    parser.add_option(
        "--simulations",
        dest="simulations",
        type="int",
        default=1000,
        metavar="POSITIONS",
    )
    parser.add_option(
        "--size",
        dest="size",
        type="int",
        default=3,
        metavar="SIZE",
    )

    opts, args = parser.parse_args(args)

    p = tak.Position.from_config(tak.Config(size=opts.size))

    engine = mcts.MCTS(mcts.Config(simulation_limit=opts.simulations))

    engine.get_move(p)


if __name__ == "__main__":
    main(sys.argv)
