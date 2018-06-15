import tak.ptn
import tak.proto

import attr
import os
import struct

import numpy as np

import tensorflow as tf

def xread(fh, n):
  b = fh.read(n)
  if len(b) == 0:
    raise EOFError

  if len(b) != n:
    raise IOError("incomplete read ({0}/{1} at off={2})".format(
      len(b), n, fh.tell()))
  return b

def load_proto(path):
  positions = []
  with open(path, 'rb') as f:
    while True:
      try:
        rlen, = struct.unpack(">L", xread(f, 4))
        positions.append(tak.proto.CorpusEntry.FromString(xread(f, rlen)))
      except EOFError:
        break
  return positions

def write_proto(path, positions):
  with open(path, 'wb') as f:
    for rec in positions:
      data = rec.SerializeToString()
      f.write(struct.pack(">L", len(data)))
      f.write(data)

def to_features(positions, add_symmetries=False):
  p = tak.ptn.parse_tps(positions[0].tps)
  size = p.size
  feat = tak.train.Featurizer(size)

  def gen():
    for pos in positions:
      count = 8 if add_symmetries else 1
      xs = np.zeros((count,) + feat.feature_shape())
      ys = np.zeros((count, feat.move_count()))

      p = tak.ptn.parse_tps(pos.tps)
      m = tak.ptn.parse_move(pos.move)
      if add_symmetries:
        feat.features_symmetries(p, out=xs)
        for j,sym in enumerate(tak.symmetry.SYMMETRIES):
          ys[j][feat.move2id(tak.symmetry.transform_move(sym, m, size))] = 1
      else:
        feat.features(p, out=xs[0])
        ys[0][tak.train.move2id(m, size)] = 1
      for (x, y) in zip(xs, ys):
        yield {
          'position': x,
          'move': y,
        }
  return tf.data.Dataset.from_generator(
    gen,
    {
      'position': tf.float32,
      'move': tf.float32,
    },
    {
      'position': feat.feature_shape(),
      'move': (feat.move_count(),),
    })

def raw_load(dir):
  return (
    load_proto(os.path.join(dir, 'train.dat')),
    load_proto(os.path.join(dir, 'test.dat')),
  )


def load_features(dir, add_symmetries=False):
  train, test = raw_load(dir)
  return (
    to_features(train, add_symmetries),
    to_features(test, add_symmetries))

__all__ = ['load_proto', 'write_proto', 'load_features']
