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
