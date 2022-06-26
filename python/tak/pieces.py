import enum

from attrs import define


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


_piece_cache = [[None for k in Kind] for c in Color]


@define(frozen=True, slots=True)
class Piece(object):
    color: Color
    kind: Kind

    def is_road(self):
        return self.kind.is_road()

    @classmethod
    def _init_cache(cls):
        for c in Color:
            for k in Kind:
                _piece_cache[c.value][k.value] = cls(c, k)

    @classmethod
    def cached(self, color, kind):
        return _piece_cache[color.value][kind.value]


Piece._init_cache()

__all__ = ["Color", "Kind", "Piece"]
