import tak.symmetry
import tak.ptn

W  = tak.Piece(tak.Color.WHITE, tak.Kind.FLAT)
WC = tak.Piece(tak.Color.WHITE, tak.Kind.CAPSTONE)
WS = tak.Piece(tak.Color.WHITE, tak.Kind.STANDING)

B  = tak.Piece(tak.Color.BLACK, tak.Kind.FLAT)
BC = tak.Piece(tak.Color.BLACK, tak.Kind.CAPSTONE)
BS = tak.Piece(tak.Color.BLACK, tak.Kind.STANDING)

class TestSymmetries(object):
  def test_empty(self):
    p = tak.Position.from_config(tak.Config(size=5))
    syms = tak.symmetry.symmetries(p)
    assert len(syms) == 1
    assert syms[0][1] == p

  def test_four(self):
    p = tak.Position.from_squares(
      tak.Config(size=5),
      [
        [W], [], [], [], [],
        [], [], [], [], [],
        [], [], [], [], [],
        [], [], [], [], [],
        [], [], [], [], [],
      ],
      2
    )

    a = tak.Position.from_squares(
      tak.Config(size=5),
      [
        [], [], [], [], [],
        [], [], [], [], [],
        [], [], [], [], [],
        [], [], [], [], [],
        [W], [], [], [], [],
      ],
      2
    )
    b = tak.Position.from_squares(
      tak.Config(size=5),
      [
        [], [], [], [], [W],
        [], [], [], [], [],
        [], [], [], [], [],
        [], [], [], [], [],
        [], [], [], [], [],
      ],
      2
    )
    c = tak.Position.from_squares(
      tak.Config(size=5),
      [
        [], [], [], [], [],
        [], [], [], [], [],
        [], [], [], [], [],
        [], [], [], [], [],
        [], [], [], [], [W],
      ],
      2
    )

    syms = tak.symmetry.symmetries(p)
    assert len(syms) == 4
    ps = [p for (s,p) in syms]
    assert a in ps
    assert b in ps
    assert c in ps

  def test_eight(self):
    p = tak.Position.from_squares(
      tak.Config(size=5),
      [
        [W], [], [], [], [],
        [B], [], [], [], [],
        [ ], [], [], [], [],
        [ ], [], [], [], [],
        [ ], [], [], [], [],
      ],
      2
    )
    syms = tak.symmetry.symmetries(p)
    assert len(syms) == 8
    assert syms[0][1] == p

class TestTransformMove(object):
  def test_transform_move(self):
    move = tak.ptn.parse_move('4a1>')
    syms = [tak.symmetry.transform_move(s, move, 5)
            for s in tak.symmetry.SYMMETRIES]
    assert (
      set([tak.ptn.format_move(m) for m in syms])
      ==
      set(['4a1>', '4a1+', '4e1+', '4e1<', '4a5-', '4a5>', '4e5-', '4e5<'])
    )
