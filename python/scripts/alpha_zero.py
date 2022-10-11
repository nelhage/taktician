#!/usr/bin/env python
import os.path

import yaml
from tak.alphazero import cli, trainer


def main():
    args = cli.build_parser().parse_args()

    if args.run_dir and os.path.exists(os.path.join(args.run_dir, "run.yaml")):
        with open(os.path.join(args.run_dir, "run.yaml")) as fh:
            config = yaml.unsafe_load(fh)

    else:
        config = cli.build_train_run(args)

    train = trainer.TrainingRun(config=config)
    train.run()


if __name__ == "__main__":
    main()
