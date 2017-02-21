import attr

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

@attr.s(frozen=True, slots=True)
class Position(object):
  size = attr.ib()

  stones      = attr.ib()

  ply = attr.ib()

  board    = attr.ib()

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

    return attr.assoc(self, **delta)

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
    newstones = attr.assoc(cs, **{slot: getattr(cs, slot) - 1})

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

__all__ = ['Config', 'StoneCounts', 'Position', 'IllegalMove']
