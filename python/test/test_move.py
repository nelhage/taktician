import tak


class TestMove(object):
    def test_is_slide(self):
        assert not tak.MoveType.PLACE_FLAT.is_slide()
        assert not tak.MoveType.PLACE_STANDING.is_slide()
        assert not tak.MoveType.PLACE_CAPSTONE.is_slide()
        assert tak.MoveType.SLIDE_LEFT.is_slide()
        assert tak.MoveType.SLIDE_RIGHT.is_slide()
        assert tak.MoveType.SLIDE_UP.is_slide()
        assert tak.MoveType.SLIDE_DOWN.is_slide()

    def test_direction(self):
        assert tak.MoveType.SLIDE_LEFT.direction() == (-1, 0)
        assert tak.MoveType.SLIDE_RIGHT.direction() == (1, 0)
        assert tak.MoveType.SLIDE_UP.direction() == (0, 1)
        assert tak.MoveType.SLIDE_DOWN.direction() == (0, -1)


def test_all_moves():
    moves = set(tak.enumerate_moves(5))

    assert tak.Move(0, 0) in moves
    assert tak.Move(0, 4) in moves
    assert tak.Move(4, 4) in moves

    assert tak.Move(0, 0, tak.MoveType.PLACE_CAPSTONE) in moves
    assert tak.Move(0, 0, tak.MoveType.PLACE_STANDING) in moves

    assert tak.Move(5, 5) not in moves

    assert tak.Move(0, 0, tak.MoveType.SLIDE_RIGHT, (1, 1, 1, 1)) in moves
    assert tak.Move(1, 0, tak.MoveType.SLIDE_RIGHT, (1, 1, 1)) in moves
    assert tak.Move(2, 0, tak.MoveType.SLIDE_RIGHT, (1, 1, 1)) not in moves

    assert all(m.type != tak.MoveType.SLIDE_LEFT for m in moves if m.x == 0)
    assert all(m.type != tak.MoveType.SLIDE_RIGHT for m in moves if m.x == 4)
    assert all(m.type != tak.MoveType.SLIDE_UP for m in moves if m.y == 4)
    assert all(m.type != tak.MoveType.SLIDE_DOWN for m in moves if m.y == 0)
