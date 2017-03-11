import tak.model
import tak.ptn

import argparse
import sys
import csv

import numpy as np

FLAGS = None

def main(args):
  model = tak.model.load_model(FLAGS.model,
                               eval_symmetries=FLAGS.symmetries)

  positions = tak.train.load_positions(FLAGS.corpus)

  ok1 = 0
  ok5 = 0
  tot = 0

  for p, m in positions:
    probs = model.evaluate(p)
    mid = tak.train.move2id(m, 5)
    top = np.flip(np.argsort(probs), axis=0)
    ok1  += top[0] == mid
    ok5 += mid in top[:5]
    tot += 1

  print("{0}/{1}/{2}: {3:.2f}% / {4:.2f}%".format(
    ok1, ok5, tot, 100*ok1/tot, 100*ok5/tot))

def arg_parser():
  parser = argparse.ArgumentParser()
  parser.add_argument('--model',
                      type=str,
                      default=None,
                      required=True,
                      help='model to run')

  parser.add_argument('--symmetries',
                      default=False,
                      action='store_true',
                      help='average over all symmetries')

  parser.add_argument('--corpus', type=str,
                      required=True)
  return parser


if __name__ == '__main__':
  parser = arg_parser()
  FLAGS, unparsed = parser.parse_known_args()
  main(unparsed)
