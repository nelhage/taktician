import attr
import enum

@enum.unique
class MoveType(enum.Enum):
  PLACE_FLAT     = 1
  PLACE_STANDING = 2
  PLACE_CAPSTONE = 3
  SLIDE_LEFT     = 4
  SLIDE_RIGHT    = 5
  SLIDE_UP       = 6
  SLIDE_DOWN     = 7

  def is_slide(self):
    return self.value >= MoveType.SLIDE_LEFT.value

  def direction(self):
    assert(self.is_slide())

    if self == MoveType.SLIDE_LEFT:
      return (-1, 0)
    if self == MoveType.SLIDE_RIGHT:
      return (1, 0)
    if self == MoveType.SLIDE_UP:
      return (0, 1)
    if self == MoveType.SLIDE_DOWN:
      return (0, -1)

    assert(False)

@attr.s(frozen=True, slots=True)
class Move(object):
  x      = attr.ib(validator = attr.validators.instance_of(int))
  y      = attr.ib(validator = attr.validators.instance_of(int))
  type   = attr.ib(validator = attr.validators.instance_of(MoveType),
                   default = MoveType.PLACE_FLAT)
  slides = attr.ib(validator = attr.validators.instance_of(list),
                   default = attr.Factory(list))

__all__ = ['MoveType', 'Move']
