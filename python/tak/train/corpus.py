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
  results = attr.ib()

  def minibatches(self, batch_size):
    perm = np.random.permutation(len(self.positions))
    i = 0
    while i < len(self.positions):
      yield (self.positions[perm[i:i+batch_size]],
             self.moves[perm[i:i+batch_size]],
             self.results[perm[i:i+batch_size]],
      )
      i += batch_size

def parse_result(r):
  if r == '':
    return None
  return float(r)

def load_positions(path, require_result=False):
  positions = []

  with open(path) as f:
    reader = csv.reader(f)
    for row in reader:
      tps, m, result = row[:3]
      if require_result and result == '':
        continue
      positions.append((
        tak.ptn.parse_tps(tps),
        tak.ptn.parse_move(m),
        parse_result(result),
      ))
  return positions

def load_corpus_file(path, add_symmetries=False, require_result=False):
  positions = load_positions(path, require_result=require_result)
  size = positions[0][0].size
  feat = tak.train.Featurizer(size)

  count = len(positions)
  if add_symmetries:
    count *= 8
  xs = np.zeros((count,) + feat.feature_shape())
  ys = np.zeros((count, feat.move_count()))
  rs = np.zeros((count,))

  for i, (p, m, r) in enumerate(positions):
    if add_symmetries:
      feat.features_symmetries(p, out=xs[8*i:8*(i+1)])
      for j,sym in enumerate(tak.symmetry.SYMMETRIES):
        ys[8*i + j][feat.move2id(tak.symmetry.transform_move(sym, m, size))] = 1
        if r is not None:
          rs[8*i + j] = r
    else:
      feat.features(p, out=xs[i])
      ys[i][tak.train.move2id(m, size)] = 1
      if r is not None:
        rs[i] = r
  return Dataset(size, xs, ys, rs)

def load_corpus(dir, add_symmetries=False, require_result=False):
  return (
    load_corpus_file(os.path.join(dir, 'train.csv'), add_symmetries, require_result),
    load_corpus_file(os.path.join(dir, 'test.csv'), add_symmetries, require_result))

__all__ = ['Dataset', 'load_corpus_file', 'load_corpus', 'load_positions']
