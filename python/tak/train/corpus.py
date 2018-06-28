import tak.ptn
import tak.proto

import attr
import os
import struct
import grpc

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

def to_features(positions, add_symmetries=False, stub=None):
  p = tak.ptn.parse_tps(positions[0].tps)
  size = p.size
  feat = tak.train.Featurizer(size)
  if stub is None:
    channel = grpc.insecure_channel('localhost:55430')
    stub = tak.proto.TakticianStub(channel)

  def gen():
    for pos in positions:
      p = tak.ptn.parse_tps(pos.tps)
      m = tak.ptn.parse_move(pos.move)
      if pos.in_tak != pos.UNSET:
        is_tak = pos.in_tak == pos.IN_TAK
      else:
        is_tak = stub.IsPositionInTak(
          tak.proto.IsPositionInTakRequest(position=pos.tps)).inTak
      if add_symmetries:
        ps = [tak.symmetry.transform_position(sym, p)
                     for sym in tak.symmetry.SYMMETRIES]
        ms = [tak.symmetry.transform_move(sym, m, size)
                     for sym in tak.symmetry.SYMMETRIES]
      else:
        ps = [p]
        ms = [m]
      for (p, m) in zip(ps, ms):
        onehot = np.zeros((feat.move_count(),))
        onehot[feat.move2id(m)] = 1.0
        yield {
          'position': feat.features(p),
          'move': onehot,
          'is_tak': is_tak,
        }

  return tf.data.Dataset.from_generator(
    gen,
    {
      'position': tf.float32,
      'move': tf.float32,
      'is_tak': tf.float32,
    },
    {
      'position': feat.feature_shape(),
      'move': (feat.move_count(),),
      'is_tak': (),
    })

def raw_load(dir):
  return (
    load_proto(os.path.join(dir, 'train.dat')),
    load_proto(os.path.join(dir, 'test.dat')),
  )

def load_dataset(path, size):
  feat = tak.train.Featurizer(size)
  features = {
    'position': tf.FixedLenFeature(shape=feat.feature_shape(), dtype=tf.float32),
    'move': tf.FixedLenFeature(shape=(feat.move_count()), dtype=tf.float32),
    'is_tak': tf.FixedLenFeature(shape=(), dtype=tf.float32),
  }
  def _parse(examples):
    return tf.parse_single_example(examples, features)

  return (
    tf.data.TFRecordDataset([path])
    .map(_parse)
  )

def load_corpus(dir, add_symmetries=False):
  train, test = raw_load(dir)
  return (
    to_features(train, add_symmetries),
    to_features(test, add_symmetries))

def load_features(dir, size):
  return (
    load_dataset(os.path.join(dir, "train.tfrecord"), size),
    load_dataset(os.path.join(dir, "test.tfrecord"), size),
  )

__all__ = ['load_proto', 'write_proto',
           'load_dataset',
           'load_corpus', 'load_features']
