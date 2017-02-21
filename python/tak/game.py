import attr

from . import moves

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

  def move(self, m):
    delta = {
      'move': self.move + 1,
    }

    place = None
    dx,dy = None

    if m.type.is_slide():
      self._move_slide(m, delta)
    else:
      self._move_place(m, delta)

    return attr.assoc(self, delta)

  def _move_place(self, m, delta):
    if self.ply < 2 and m.type != moves.MoveType.PLACE_FLAT:
      raise IllegalMove("Illegal opening")

  def _move_slide(self, m, delta):
    if self.ply < 2:
      raise IllegalMove("Illegal opening")


class IllegalMove(Exception):
  pass

__all__ = ['Config', 'Position', 'IllegalMove']
