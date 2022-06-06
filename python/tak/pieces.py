import enum
import attr


class Color(enum.Enum):
    WHITE = 0
    BLACK = 1

    def flip(self):
        return Color(1 - self.value)


class Kind(enum.Enum):
    FLAT = 0
    STANDING = 1
    CAPSTONE = 2

    def is_road(self):
        return self == Kind.FLAT or self == Kind.CAPSTONE


@attr.s(frozen=True, slots=True)
class Piece(object):
    color = attr.ib(validator=attr.validators.instance_of(Color))
    kind = attr.ib(validator=attr.validators.instance_of(Kind))

    def is_road(self):
        return self.kind.is_road()

    @classmethod
    def __call__(cls, color, kind):
        return _piece_cache[color.value][kind.value]


_piece_cache = [[Piece(c, p) for p in Kind] for c in Color]


def Piece(color, kind):
    return _piece_cache[color.value][kind.value]


__all__ = ["Color", "Kind", "Piece"]
