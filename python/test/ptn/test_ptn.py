import tak
import tak.ptn

import pytest

class TestParseMove(object):
  def test_valid(self):
    cases = [
      (
        "a1",
        tak.Move(0, 0, tak.MoveType.PLACE_FLAT),
        "a1",
      ),
      (
        "Sa4",
        tak.Move(0, 3, tak.MoveType.PLACE_STANDING),
        "Sa4",
      ),
      (
        "Ch7",
        tak.Move(7, 6, tak.MoveType.PLACE_CAPSTONE),
        "Ch7",
      ),
      (
        "Fh7",
        tak.Move(7, 6, tak.MoveType.PLACE_FLAT),
        "h7",
      ),
      (
        "a1>",
        tak.Move(0, 0, tak.MoveType.SLIDE_RIGHT, [1]),
        "a1>",
      ),
      (
        "2a2<",
        tak.Move(0, 1, tak.MoveType.SLIDE_LEFT, [2]),
        "2a2<",
      ),
      (
        "3a1+111",
        tak.Move(0, 0, tak.MoveType.SLIDE_UP, [1, 1, 1]),
        "3a1+111",
      ),
#      (
#        "5d4-22",
#        tak.Move(3, 3, tak.MoveType.SlideDown, Slides: tak.MkSlides(2, 2, 1)),
#        "5d4-221",
#      ),
    ]

    for case in cases:
      ptn, move, out = case
      parsed = tak.ptn.parse_move(ptn)
      assert parsed == move, "parse_ptn('{0}') = {1} != {2}".format(
        ptn, parsed, move)

  def test_invalid(self):
    cases = [
      "",
      "a11",
      "z3",
      "14c4>",
      "6a1",
      "6a1>2222",
      "a",
      "3a",
      "5d4-22",
    ]
    for tc in cases:
      with pytest.raises(tak.ptn.BadMove):
        assert tak.ptn.parse_move(tc) == None, "parsing {0}".format(tc)
