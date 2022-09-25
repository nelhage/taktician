import enum
import typing as T

from attrs import define, field


@enum.unique
class MoveType(enum.Enum):
    PLACE_FLAT = 1
    PLACE_STANDING = 2
    PLACE_CAPSTONE = 3
    SLIDE_LEFT = 4
    SLIDE_RIGHT = 5
    SLIDE_UP = 6
    SLIDE_DOWN = 7

    def is_slide(self):
        return self.value >= MoveType.SLIDE_LEFT.value

    def direction(self):
        assert self.is_slide()
        return DIRECTIONS[self]

    @staticmethod
    def from_direction(dx, dy):
        return RDIRECTIONS[(dx, dy)]

    def __lt__(self, rhs):
        return self.value < rhs.value


DIRECTIONS = {
    MoveType.SLIDE_LEFT: (-1, 0),
    MoveType.SLIDE_RIGHT: (1, 0),
    MoveType.SLIDE_UP: (0, 1),
    MoveType.SLIDE_DOWN: (0, -1),
}
RDIRECTIONS = dict((v, k) for (k, v) in DIRECTIONS.items())


@define(frozen=True)
class Move(object):
    x: int
    y: int
    type: MoveType = field(default=MoveType.PLACE_FLAT)
    slides: T.Optional[tuple[int]] = None


ALL_SLIDES = [() for i in range(9)]


def _compute_slides(size):
    slides = []
    for i in range(1, size + 1):
        slides.append((i,))
        for inner in ALL_SLIDES[size - i]:
            slides.append((i,) + inner)
    return slides


for s in range(1, 9):
    ALL_SLIDES[s] = _compute_slides(s)


def all_moves_for_size(size):
    out = []
    for x in range(size):
        for y in range(size):
            out.append(Move(x, y, MoveType.PLACE_FLAT))
            out.append(Move(x, y, MoveType.PLACE_STANDING))
            out.append(Move(x, y, MoveType.PLACE_CAPSTONE))

            dirs = [
                (MoveType.SLIDE_LEFT, x),
                (MoveType.SLIDE_RIGHT, size - x - 1),
                (MoveType.SLIDE_DOWN, y),
                (MoveType.SLIDE_UP, size - y - 1),
            ]
            for slide in ALL_SLIDES[size]:
                for d, l in dirs:
                    if len(slide) <= l:
                        out.append(Move(x, y, d, slide))
    return out


__all__ = ["MoveType", "Move", "ALL_SLIDES", "all_moves_for_size"]
