import attr
import enum

from . import moves
from . import pieces

@attr.s(frozen=True, slots=True)
class Config(object):
  size   = attr.ib(validator = attr.validators.instance_of(int))
  pieces = attr.ib(validator = attr.validators.optional(attr.validators.instance_of(int)),
                   default = None)
  capstones = attr.ib(validator = attr.validators.optional(attr.validators.instance_of(int)),
                      default = None)

  @property
  def flat_count(self):
    if self.pieces is not None:
      return self.pieces
    return self.DEFAULT_PIECES[self.size]

  @property
  def capstone_count(self):
    if self.capstones is not None:
      return self.capstones
    return self.DEFAULT_CAPS[self.size]

  DEFAULT_PIECES = [0, 0, 0, 10, 15, 21, 30, 40, 50]
  DEFAULT_CAPS   = [0, 0, 0, 0, 0, 1, 1, 1, 2]

@attr.s(frozen=True, slots=True)
class StoneCounts(object):
  stones = attr.ib()
  caps   = attr.ib()

class WinReason(enum.Enum):
  ROAD = 1
  FLATS = 2

@attr.s(frozen=True, slots=True)
class Position(object):
  size   = attr.ib()
  stones = attr.ib()
  ply    = attr.ib()
  board  = attr.ib()

  @classmethod
  def from_config(cls, config):
    size   = config.size
    stones = StoneCounts(stones = config.flat_count,
                         caps   = config.capstone_count)

    return cls(
      size = size,
      ply = 0,
      stones = (stones, stones),
      board = [[] for _ in range(size*size)]
    )

  @classmethod
  def from_squares(cls, cfg, squares, ply):
    if len(squares) != cfg.size * cfg.size:
      raise ValueError("Wrong board size")

    counts = ([0,0], [0,0])

    for sq in squares:
      for p in sq:
        if p.kind == pieces.Kind.CAPSTONE:
          counts[p.color.value][1] += 1
        else:
          counts[p.color.value][0] += 1

    stones = (StoneCounts(cfg.flat_count - counts[0][0],
                          cfg.capstone_count - counts[0][1]),
              StoneCounts(cfg.flat_count - counts[1][0],
                          cfg.capstone_count - counts[1][1]))
    return cls(
      size = cfg.size,
      ply = ply,

      stones = stones,
      board = squares,
    )

  def to_move(self):
    if self.ply % 2 == 0:
      return pieces.Color.WHITE
    return pieces.Color.BLACK

  def in_bounds(self, x, y):
    return (x >= 0 and x < self.size and
            y >= 0 and y < self.size)

  def winner(self):
    color = self.has_road()
    if color is not None:
      return (color, WinReason.ROAD)
    if all(self.board) or any((s.stones+s.caps) == 0 for s in self.stones):
      return (self.flats_winner(), WinReason.FLATS)
    return (None, None)

  def is_road(self, x, y):
    sq = self[x,y]
    return len(sq) > 0 and sq[0].is_road()

  def _walk(self, seeds, color, horiz):
    seen = set()
    q = list(seeds)
    while q:
      j = q.pop()
      if j in seen:
        continue
      seen.add(j)
      x, y = j
      if not self.in_bounds(x,y):
        continue

      if not self.is_road(x, y) or self[x,y][0].color != color:
        continue

      if horiz and x == self.size - 1:
        return True
      if (not horiz) and y == self.size - 1:
        return True

      q.append((x+1, y))
      q.append((x-1, y))
      q.append((x, y+1))
      q.append((x, y-1))

    return False

  def has_road(self):
    left = [(0, i) for i in range(self.size)]
    top = [(i, 0) for i in range(self.size)]

    w = (self._walk(left, pieces.Color.WHITE, True) or
         self._walk(top, pieces.Color.WHITE, False))
    b = (self._walk(left, pieces.Color.BLACK, True) or
         self._walk(top, pieces.Color.BLACK, False))
    if w and b:
      return self.to_move().flip()
    if w:
      return pieces.Color.WHITE
    if b:
      return pieces.Color.BLACK
    return None

  def flat_counts(self):
    w, b = 0, 0
    for sq in self.board:
      if len(sq) == 0 or sq[0].kind != pieces.Kind.FLAT:
        continue
      if sq[0].color == pieces.Color.WHITE:
        w += 1
      else:
        b += 1
    return (w, b)

  def flats_winner(self):
    w,b = self.flat_counts()
    if w > b:
      return pieces.Color.WHITE
    if w < b:
      return pieces.Color.BLACK
    return self.to_move().flip()

  def __getitem__(self, pos):
    x,y = pos
    return self.board[y * self.size + x]

  def move(self, m):
    delta = {
      'ply': self.ply + 1,
    }

    if m.type.is_slide():
      self._move_slide(m, delta)
    else:
      self._move_place(m, delta)

    return attr.evolve(self, **delta)

  def _move_place(self, m, delta):
    if self.ply < 2 and m.type != moves.MoveType.PLACE_FLAT:
      raise IllegalMove("Illegal opening")
    if self[m.x,m.y]:
      raise IllegalMove("Place on an occupied square")
    color = self.to_move()
    if self.ply < 2:
      color = color.flip()

    slot = 'stones'
    kind = pieces.Kind.FLAT
    if m.type == moves.MoveType.PLACE_CAPSTONE:
      slot = 'caps'
      kind = pieces.Kind.CAPSTONE
    elif m.type == moves.MoveType.PLACE_STANDING:
      kind = pieces.Kind.STANDING

    cs = self.stones[color.value]
    if getattr(cs, slot) <= 0:
      raise IllegalMove("not enough stones")
    newstones = attr.evolve(cs, **{slot: getattr(cs, slot) - 1})

    if color == pieces.Color.WHITE:
      delta['stones'] = (newstones, self.stones[1])
    else:
      delta['stones'] = (self.stones[0], newstones)

    newboard = list(self.board)
    newboard[m.x + m.y*self.size] = [pieces.Piece(color=color, kind=kind)]
    delta['board'] = newboard

  def _move_slide(self, m, delta):
    if self.ply < 2:
      raise IllegalMove("Illegal opening")

    stack = self[m.x, m.y]
    ndrop = sum(m.slides)

    if ndrop > self.size or len(stack) < ndrop:
      raise IllegalMove("picking up too many pieces")

    if ndrop < 1:
      raise IllegalMove("must pick up at least one stone")

    if stack[0].color != self.to_move():
      raise IllegalMove("can't move opponent's stack")

    newboard = list(self.board)
    delta['board'] = newboard

    x, y = m.x, m.y
    dx, dy = m.type.direction()
    carry = stack[:ndrop]

    newboard[x + y *self.size] = stack[ndrop:]
    for drop in m.slides:
      x += dx
      y += dy
      if not self.in_bounds(x,y):
        raise IllegalMove("slide out of bounds")
      i = x + y * self.size
      orig = self.board[i]
      if len(orig) > 0 and orig[0].kind == pieces.Kind.CAPSTONE:
        raise IllegalMove("slide onto a capstone")
      if len(orig) > 0 and orig[0].kind == pieces.Kind.STANDING:
        if carry[0].kind != pieces.Kind.CAPSTONE or len(carry) != 1:
          raise IllegalMove("slide onto a standing stone")
        orig = [pieces.Piece(orig[0].color, pieces.Kind.FLAT)] + orig[1:]

      newboard[i] = carry[-drop:] + orig
      carry = carry[:-drop]


class IllegalMove(Exception):
  pass

__all__ = ['Config', 'StoneCounts', 'Position', 'IllegalMove', 'WinReason']
