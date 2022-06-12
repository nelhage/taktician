import sys

import tak.train


def main(args):
    tak.train.load_corpus(args[1], True)


if __name__ == "__main__":
    main(sys.argv)
