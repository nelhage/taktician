import tak

_moves    = [None] * 9
_move_ids = [None] * 9

def _compute_moves(size):
  if _moves[size] is not None:
    return
  moves = list(sorted(tak.enumerate_moves(size)))
  ids = dict((m, i) for (i,m) in enumerate(moves))
  _moves[size] = moves
  _move_ids[size] = ids

def move2id(move, size):
  _compute_moves(size)
  return _move_ids[size][move]

def move_count(size):
  _compute_moves(size)
  return len(_moves[size])

def id2move(id, size):
  _compute_moves(size)
  return _moves[size][id]
