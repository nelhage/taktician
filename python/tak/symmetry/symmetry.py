import tak

import numpy as np

rot = np.array(
  [
    [0, 1, 0],
    [-1, 0, 1],
    [0, 0, 1]
  ],
  dtype=np.int,
)
flip = np.array(
  [
    [-1, 0, 1],
    [0, 1, 0],
    [0, 0, 1],
  ],
  dtype=np.int,
)

SYMMETRIES = [
  np.matmul(l,r) for
  l in [
    np.identity(3, dtype=np.int),
    rot,
    np.matmul(rot, rot),
    np.matmul(np.matmul(rot, rot), rot),
  ]
  for r in [
      np.identity(3, dtype=np.int),
      flip
  ]
]

assert all(abs(np.linalg.det(m)) == 1 for m in SYMMETRIES)

def transform_position(sym, pos):
  ix = np.stack([
    np.repeat(np.arange(pos.size), pos.size),
    np.tile(np.arange(pos.size), pos.size),
    (pos.size-1)*np.ones(pos.size*pos.size)
  ], axis=-1)
  ix = np.transpose(np.matmul(sym, np.transpose(ix))).astype(np.int)
  ix = ix.reshape((pos.size, pos.size, 3))

  sqs = list(pos.board)
  for i in range(pos.size):
    for j in range(pos.size):
      oi, oj, _ = ix[i,j]
      sqs[oi + oj*pos.size] = pos[i, j]
  return tak.Position.from_squares(
    tak.Config(size = pos.size),
    sqs,
    pos.ply,
  )

def transform_move(sym, move, size):
  ox, oy, _ = np.matmul(sym, [move.x, move.y, size - 1])
  type = move.type
  if type.is_slide():
    dx, dy, _ = np.matmul(sym, move.type.direction() + (0,))
    type = tak.MoveType.from_direction(dx, dy)
  return tak.Move(int(ox), int(oy), type, move.slides)

def symmetries(pos):
  out = []
  for s in SYMMETRIES:
    t = transform_position(s, pos)
    if all(t != p for _,p in out):
      out.append((s, t))

  return out

__all__ = ['SYMMETRIES', 'transform_position', 'transform_move', 'symmetries']
