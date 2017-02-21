import tak
import pytest

def test_new():
  g = tak.Position.from_config(tak.Config(size=5))
  assert g.size == 5
  assert g.ply == 0
  assert g.whiteStones == 21
  assert g.blackStones == 21
  assert g.whiteCaps == 1
  assert g.blackCaps == 1

  assert g.to_move() == tak.Color.WHITE

class TestMove(object):
  def test_place_flat(self):
    g = tak.Position.from_config(tak.Config(size=5))
    g1 = g.move(tak.Move(
      x = 0,
      y = 0,
    ))
    assert g.ply == 0
    assert g[0,0] == []
    assert g1[0,0] == [tak.Piece(tak.Color.BLACK, tak.Kind.FLAT)]
    assert g1.ply == 1

    g2 = g1.move(tak.Move(
      x = 4,
      y = 4,
    ))

    assert g2[0,0] == [tak.Piece(tak.Color.BLACK, tak.Kind.FLAT)]
    assert g2[4,4] == [tak.Piece(tak.Color.WHITE, tak.Kind.FLAT)]

    g3 = g2.move(tak.Move(
      x = 2,
      y = 2,
    ))

    assert g3[2,2] == [tak.Piece(tak.Color.WHITE, tak.Kind.FLAT)]

  def test_initial_special(self):
    g = tak.Position.from_config(tak.Config(size=5))
    with pytest.raises(tak.IllegalMove):
      g.move(tak.Move(0, 0, tak.MoveType.PLACE_CAPSTONE))

    with pytest.raises(tak.IllegalMove):
      g.move(tak.Move(0, 0, tak.MoveType.PLACE_STANDING))
