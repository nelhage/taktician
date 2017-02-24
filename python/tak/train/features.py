import tak

import enum

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

def feature_shape(size):
  # (x, y, planes)
  return (size, size, 2 * int(1.5*size) + EXTRA_PLANES)

def clamp(n, lim):
  if n < 0:
    return 0
  if n >= lim:
    return lim-1
  return n

def features(pos, buf=None):
  if buf is None:
    buf = np.zeros(feature_shape(pos.size))
  else:
    buf[:] = 0

  me = pos.to_move()
  max_depth = int(1.5 * pos.size)
  extra = buf[:,:,2 * max_depth:]

  for i in range(pos.size):
    for j in range(pos.size):
      sq = pos[i,j]
      for k,s in enumerate(sq[:max_depth]):
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
