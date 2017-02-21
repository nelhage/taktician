import tak

def test_new():
  g = tak.Position.from_config(tak.Config(size=5))
  assert g.size == 5
  assert g.ply == 0
  assert g.whiteStones == 21
  assert g.blackStones == 21
  assert g.whiteCaps == 1
  assert g.blackCaps == 1
