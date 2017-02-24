import tak.ptn

import os
import csv
import attr

import numpy as np

@attr.s(frozen=True)
class Dataset(object):
  size = attr.ib()
  positions = attr.ib()
  moves = attr.ib()

  def minibatches(self, batch_size):
    perm = np.random.permutation(len(self.positions))
    i = 0
    while i < len(self.positions):
      yield (self.positions[perm[i:i+batch_size]], self.moves[perm[i:i+batch_size]])
      i += batch_size

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
  return Dataset(size, xs, ys)

def load_corpus(dir):
  return (
    load_corpus_file(os.path.join(dir, 'train.csv')),
    load_corpus_file(os.path.join(dir, 'test.csv')))

__all__ = ['Dataset', 'load_corpus_file', 'load_corpus']
