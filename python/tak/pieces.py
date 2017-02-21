import enum
import attr

class Color(enum.Enum):
  WHITE = 0
  BLACK = 1

  def flip(self):
    return Color(1-self.value)

class Kind(enum.Enum):
  FLAT     = 0
  STANDING = 1
  CAPSTONE = 2

@attr.s(frozen=True, slots=True)
class Piece(object):
  color = attr.ib(validator = attr.validators.instance_of(Color))
  kind  = attr.ib(validator = attr.validators.instance_of(Kind))


__all__ = ['Color', 'Kind', 'Piece']
