import sys

import tak.ptn
import tak.symmetry


def main(args):
    p = tak.ptn.parse_tps(args[1])
    for i in range(1000):
        tak.symmetry.symmetries(p)


if __name__ == "__main__":
    main(sys.argv)
