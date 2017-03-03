import tak
import tak.symmetry

from . import moves

import enum
import functools

import numpy as np

# (1, 2, ..., K, > K)
RESERVES_PLANES = 7

# we record my flat count advantage
# (< -K, ..., -1, 0, 1, ... > K)
FLATS_PLANES    = 7
MAX_FLAT_DELTA  = (FLATS_PLANES // 2)

class FeaturePlane(object):
  CAPSTONE  = 0
  STANDING  = 1
  ONES      = 2
  ZEROS     = 3

  MY_RESERVES    = 4
  MY_RESERVES_MAX = MY_RESERVES + RESERVES_PLANES
  MY_CAPS        = MY_RESERVES_MAX
  THEIR_RESERVES = MY_CAPS + 1
  THEIR_RESERVES_MAX = THEIR_RESERVES + RESERVES_PLANES
  THEIR_CAPS     = THEIR_RESERVES_MAX
  FLATS          = THEIR_CAPS + 1
  FLATS_MAX      = FLATS + FLATS_PLANES
  MAX            = FLATS_MAX

EXTRA_PLANES = FeaturePlane.MAX

def clamp(n, lim):
  if n < 0:
    return 0
  if n >= lim:
    return lim-1
  return n

@functools.lru_cache()
def _compute_perms(size):
  identity = np.transpose(
    np.stack([
      np.repeat(np.arange(size), size),
      np.tile(np.arange(size), size),
      np.full((size*size,), size-1, dtype=np.intp),
    ], axis=-1))
  symmetry_perms = []
  move_perms = []

  for sym in tak.symmetry.SYMMETRIES:
    ix = np.transpose(np.matmul(np.linalg.inv(sym), identity)).astype(np.int)
    symmetry_perms.append(
      np.ravel_multi_index([ix[:,0], ix[:,1]], (size, size)))

    perm = np.ndarray(moves.move_count(size), dtype=np.intp)
    for i in range(len(perm)):
      tm = tak.symmetry.transform_move(sym, moves.id2move(i, size), size)
      tmid = moves.move2id(tm, size)
      perm[i] = tmid

    move_perms.append(perm)
  return (symmetry_perms, move_perms)

class Featurizer(object):
  def __init__(self, size):
    self.size = size
    self.precompute()

  def feature_shape(self):
    # (x, y, planes)
    return (self. size, self.size, self.stack_planes + EXTRA_PLANES)

  def precompute(self):
    self.stack_depth  = int(1.5*self.size)
    self.stack_planes = 2 * self.stack_depth
    self.symmetry_perms, self.move_perms = _compute_perms(self.size)

  def id2move(self, i):
    return moves.id2move(i, self.size)

  def move2id(self, m):
    return moves.move2id(m, self.size)

  def move_count(self):
    return moves.move_count(self.size)

  def features(self, pos, out=None):
    if pos.size != self.size:
      raise ValueError("size mismatch")

    if out is None:
      buf = np.zeros(self.feature_shape())
    else:
      buf = out
      buf[:] = 0

    me = pos.to_move()
    extra = buf[:,:,self.stack_planes:]

    for i in range(pos.size):
      for j in range(pos.size):
        sq = pos[i,j]
        for k,s in enumerate(sq[:self.stack_depth]):
          buf[i,j,2*k]     = s.color == me
          buf[i,j,2*k + 1] = s.color != me
          if k == 0:
            extra[i,j,FeaturePlane.CAPSTONE] = s.kind == tak.Kind.CAPSTONE
            extra[i,j,FeaturePlane.STANDING] = s.kind == tak.Kind.STANDING

    extra[:,:,FeaturePlane.ONES] = 1

    wf, bf = pos.flat_counts()
    df = wf - bf
    if me == tak.Color.BLACK:
      df = -df
    df = clamp(df + MAX_FLAT_DELTA, FLATS_PLANES)
    extra[:,:,FeaturePlane.FLATS + df] = 1

    extra[
      :,:, clamp(FeaturePlane.MY_RESERVES + pos.stones[me.value].stones - 1, RESERVES_PLANES)
    ] = 1
    if pos.stones[me.value].caps > 0 :
      extra[:,:, FeaturePlane.MY_CAPS] = 1
    extra[
      :,:, clamp(FeaturePlane.THEIR_RESERVES + pos.stones[me.flip().value].stones - 1, RESERVES_PLANES)
    ] = 1
    if pos.stones[me.flip().value].caps > 0 :
      extra[:,:, FeaturePlane.THEIR_CAPS] = 1
    return buf

  def features_symmetries(self, pos, out=None):
    if out is not None:
      assert out.shape == (8,) + self.feature_shape()
      feat = out
    else:
      feat = np.ndarray((8,) + self.feature_shape())

    self.features(pos, feat[0])

    sqs = self.size * self.size

    vec = feat[0].reshape((sqs, -1))

    for i in range(1,8):
      feat[i] = vec[self.symmetry_perms[i]].reshape(self.feature_shape())
    return feat

  def unpermute_moves(self, moves):
    assert moves.shape == (8, self.move_count())
    for (i, perm) in enumerate(self.move_perms):
        moves[i] = moves[i][perm]

def feature_shape(size):
  return Featurizer(size).feature_shape()

def features(pos, out=None):
  return Featurizer(pos.size).features(pos, out)
