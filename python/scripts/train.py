import tak.ptn
import tak.train

import sys
import os
import argparse
import csv

import attr

import numpy as np
import tensorflow as tf

FLAGS = None

@attr.s(frozen=True)
class Dataset(object):
  positions = attr.ib()
  moves = attr.ib()

def load_corpus_file(path):
  positions = []

  with open(path) as f:
    reader = csv.reader(f)
    for row in reader:
      tps, m = row[:2]
      positions.append((
        tak.ptn.parse_tps(tps),
        tak.ptn.parse_move(m)))

  size = positions[0][0].size

  xs = np.zeros((len(positions),) + tak.train.feature_shape(size))
  ys = np.zeros((len(positions), tak.train.move_count(size)))

  for i, (p, m) in enumerate(positions):
    tak.train.features(p, xs[i])
    ys[i][tak.train.move2id(m, size)] = 1
  return Dataset(xs, ys)

def load_corpus(dir):
  return (
    load_corpus_file(os.path.join(dir, 'train.csv')),
    load_corpus_file(os.path.join(dir, 'test.csv')))

def main(args):
  print("Loading data...")
  train, test = load_corpus(FLAGS.corpus)
  print("Loaded {0} training cases and {1} test cases...".format(
    len(train.positions), len(test.positions)))

if __name__ == '__main__':
  parser = argparse.ArgumentParser()
  parser.add_argument('--corpus', type=str, default=None,
                      help='corpus to train')
  FLAGS, unparsed = parser.parse_known_args()
  tf.app.run(main=main, argv=[sys.argv[0]] + unparsed)
