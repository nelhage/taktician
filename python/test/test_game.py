import tak
import pytest

W  = tak.Piece(tak.Color.WHITE, tak.Kind.FLAT)
WC = tak.Piece(tak.Color.WHITE, tak.Kind.CAPSTONE)
WS = tak.Piece(tak.Color.WHITE, tak.Kind.STANDING)

B  = tak.Piece(tak.Color.BLACK, tak.Kind.FLAT)
BC = tak.Piece(tak.Color.BLACK, tak.Kind.CAPSTONE)
BS = tak.Piece(tak.Color.BLACK, tak.Kind.STANDING)

def test_new():
  g = tak.Position.from_config(tak.Config(size=5))
  assert g.size == 5
  assert g.ply == 0
  assert g.stones[0].stones == 21
  assert g.stones[1].stones == 21
  assert g.stones[0].caps == 1
  assert g.stones[1].caps == 1

  assert g.to_move() == tak.Color.WHITE

class TestFromStones(object):
  def test_from_stones(self):
    g = tak.Position.from_squares(
      tak.Config(size = 5),
      [ [W], [W], [B ], [W ], [W],
        [ ], [ ], [BC], [  ], [ ],
        [ ], [ ], [  ], [  ], [ ],
        [ ], [ ], [  ], [  ], [ ],
        [B], [B], [B ], [WS], [ ],
      ],
      5
    )
    assert g[0,0] == [W]
    assert g[2,1] == [BC]

    assert g.stones[tak.Color.WHITE.value].stones == 16
    assert g.stones[tak.Color.WHITE.value].caps   == 1
    assert g.stones[tak.Color.BLACK.value].stones == 17
    assert g.stones[tak.Color.BLACK.value].caps   == 0
    assert g.ply == 5

class TestPlace(object):
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
    assert g1.stones[1].caps == 1
    assert g1.stones[1].stones == 20

    g2 = g1.move(tak.Move(
      x = 4,
      y = 4,
    ))

    assert g2[0,0] == [tak.Piece(tak.Color.BLACK, tak.Kind.FLAT)]
    assert g2[4,4] == [tak.Piece(tak.Color.WHITE, tak.Kind.FLAT)]
    assert g2.stones[0].stones == 20
    assert g2.stones[1].stones == 20

    g3 = g2.move(tak.Move(
      x = 2,
      y = 2,
    ))

    assert g3[2,2] == [tak.Piece(tak.Color.WHITE, tak.Kind.FLAT)]

  def test_place_special(self):
    g = tak.Position.from_squares(
      tak.Config(size = 5),
      [[W], [ ], [ ], [ ], [ ],
       [ ], [ ], [ ], [ ], [ ],
       [ ], [ ], [ ], [ ], [ ],
       [ ], [ ], [ ], [ ], [ ],
       [ ], [ ], [ ], [ ], [B],
      ], 2)

    g1 = g.move(tak.Move(2, 2, tak.MoveType.PLACE_CAPSTONE))
    assert g1[2,2] == [WC]
    assert g1.stones[tak.Color.WHITE.value].caps == 0

    g2 = g1.move(tak.Move(1, 2, tak.MoveType.PLACE_STANDING))
    assert g2[1,2] == [BS]

    with pytest.raises(tak.IllegalMove):
      g2.move(tak.Move(2, 3, tak.MoveType.PLACE_CAPSTONE))
    with pytest.raises(tak.IllegalMove):
      g2.move(tak.Move(2, 2, tak.MoveType.PLACE_FLAT))

  def test_initial_special(self):
    g = tak.Position.from_config(tak.Config(size=5))
    with pytest.raises(tak.IllegalMove):
      g.move(tak.Move(0, 0, tak.MoveType.PLACE_CAPSTONE))

    with pytest.raises(tak.IllegalMove):
      g.move(tak.Move(0, 0, tak.MoveType.PLACE_STANDING))
