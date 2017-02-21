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

  DEFAULT_PIECES = [0, 0, 0, 10, 15, 21, 30, 40, 50]
  DEFAULT_CAPS   = [0, 0, 0, 0, 0, 1, 1, 1, 2]

@attr.s(frozen=True, slots=True)
class Position(object):
  size = attr.ib()

  whiteStones = attr.ib()
  whiteCaps   = attr.ib()
  blackStones = attr.ib()
  blackCaps   = attr.ib()

  ply = attr.ib()

  board    = attr.ib()

  @classmethod
  def from_config(cls, config):
    size   = config.size
    pieces = config.pieces
    if pieces is None:
      pieces = Config.DEFAULT_PIECES[size]
    caps = config.capstones
    if caps is None:
      caps = Config.DEFAULT_CAPS[size]

    return cls(
      size = size,

      whiteStones = pieces,
      whiteCaps   = caps,
      blackStones = pieces,
      blackCaps   = caps,

      ply = 0,

      board = [[] for _ in range(size*size)]
    )

  def to_move(self):
    if self.ply % 2 == 0:
      return pieces.Color.WHITE
    return pieces.Color.BLACK

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

    kind = pieces.Kind.FLAT
    if m.type == moves.MoveType.PLACE_CAPSTONE:
      slot = color.name.lower() + "Caps"
      kind = pieces.Kind.CAPSTONE
    else:
      slot = color.name.lower() + "Stones"
      if m.type == moves.MoveType.PLACE_STANDING:
        kind = pieces.Kind.STANDING

    if getattr(self, slot) <= 0:
      raise IllegalMove("not enough stones")
    delta[slot] = getattr(self, slot) - 1

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
    deltas['board'] = newboard

    x, y = m.x, m.y
    dx, dy = m.type.direction()
    carry = stack[:ndrop]

    newboard[x + y *self.size] = stack[ndrop:]
    for drop in m.slides:
      x += dx
      y += dy
      i = x + y * self.size
      newboard[i] = carry[-drop:] + self.board[i]
      carry = carry[:-drop]


class IllegalMove(Exception):
  pass

__all__ = ['Config', 'Position', 'IllegalMove']
