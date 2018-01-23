import tak.ptn
import tak.proto

import attr
import csv
import os
import struct

import numpy as np

@attr.s(frozen=True)
class Instance(object):
  proto    = attr.ib(
    validator = attr.validators.instance_of(tak.proto.Position))

  position = attr.ib(
    validator = attr.validators.instance_of(tak.Position))
  move = attr.ib(
    validator = attr.validators.instance_of(tak.Move))


@attr.s(frozen=True)
class Dataset(object):
  size = attr.ib()
  instances = attr.ib()

  def minibatches(self, batch_size):
    perm = np.random.permutation(len(self.instances[0]))
    i = 0
    while i < len(self):
      yield [o[perm[i:i+batch_size]] for o in self.instances]
      i += batch_size

  def __len__(self):
    return len(self.instances[0])

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
        positions.append(tak.proto.Position.FromString(xread(f, rlen)))
      except EOFError:
        break
  return positions

def load_csv(path):
  positions = []

  with open(path) as f:
    reader = csv.reader(f)
    for row in reader:
      p = tak.proto.Position()
      p.tps = row[0]
      p.move = row[1]
      if len(row) > 2 and row[2]:
        p.value = float(row[2])
      if len(row) > 3:
        p.day = row[3]
      if len(row) > 4:
        p.id = int(row[4])
      if len(row) > 5:
        p.ply = int(row[5])
      if len(row) > 6:
        p.plies = int(row[6])

      positions.append(p)
    return positions

def write_csv(path, positions):
  with open(path, 'w') as f:
    w = csv.writer(f)
    for rec in positions:
      w.writerow((rec.tps, rec.move, rec.value,
                  rec.day, rec.id, rec.ply, rec.plies,
      ))

def write_proto(path, positions):
  with open(path, 'wb') as f:
    for rec in positions:
      data = rec.SerializeToString()
      f.write(struct.pack(">L", len(data)))
      f.write(data)

def parse(positions, add_symmetries=False):
  out = []
  for p in positions:
    position = tak.ptn.parse_tps(p.tps)
    move = tak.ptn.parse_move(p.move)

    if add_symmetries:
      for sym in tak.symmetry.SYMMETRIES:
        sp = tak.symmetry.transform_position(sym, position)
        sm = tak.symmetry.transform_move(sym, move, position.size)
        out.append(Instance(
          proto = p,
          position = sp,
          move = sm))
    else:
      out.append(Instance(
        proto = p,
        position = position,
        move = move))
  return out

def to_features(positions, add_symmetries=False):
  p = tak.ptn.parse_tps(positions[0].tps)
  size = p.size
  feat = tak.train.Featurizer(size)

  count = len(positions)
  if add_symmetries:
    count *= 8
  xs = np.zeros((count,) + feat.feature_shape())
  ys = np.zeros((count, feat.move_count()))

  for i, pos in enumerate(positions):
    p = tak.ptn.parse_tps(pos.tps)
    m = tak.ptn.parse_move(pos.move)
    if add_symmetries:
      feat.features_symmetries(p, out=xs[8*i:8*(i+1)])
      for j,sym in enumerate(tak.symmetry.SYMMETRIES):
        ys[8*i + j][feat.move2id(tak.symmetry.transform_move(sym, m, size))] = 1
    else:
      feat.features(p, out=xs[i])
      ys[i][tak.train.move2id(m, size)] = 1
  return Dataset(size, (xs, ys))

def raw_load(dir):
  if os.path.isfile(os.path.join(dir, 'train.csv')):
    return (
      load_csv(os.path.join(dir, 'train.csv')),
      load_csv(os.path.join(dir, 'test.csv')),
    )
  else:
    return (
      load_proto(os.path.join(dir, 'train.dat')),
      load_proto(os.path.join(dir, 'test.dat')),
    )

def load_corpus(dir, add_symmetries=False):
  train, test = raw_load(dir)
  return (
    parse(train, add_symmetries = add_symmetries),
    parse(test, add_symmetries = add_symmetries))

def load_features(dir, add_symmetries=False):
  train, test = raw_load(dir)
  return (
    to_features(train, add_symmetries),
    to_features(test, add_symmetries))

__all__ = ['Instance', 'Dataset',
           'load_csv', 'load_proto',
           'write_csv', 'write_proto',
           'load_corpus', 'load_features']
