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
                tak.Move(0, 0, tak.MoveType.SLIDE_RIGHT, (1,)),
                "a1>",
            ),
            (
                "2a2<",
                tak.Move(0, 1, tak.MoveType.SLIDE_LEFT, (2,)),
                "2a2<",
            ),
            (
                "3a1+111",
                tak.Move(0, 0, tak.MoveType.SLIDE_UP, (1, 1, 1)),
                "3a1+111",
            ),
        ]

        for case in cases:
            ptn, move, out = case
            parsed = tak.ptn.parse_move(ptn)
            assert parsed == move, "parse_ptn('{0}') = {1} != {2}".format(
                ptn, parsed, move
            )
            rt = tak.ptn.format_move(parsed)
            assert rt == out

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


class TestParsePTN(object):
    TEST_GAME = """[Event "PTN Viewer Demo"]
[Site "Here"]
[Date "2015.11.21"]
[Player1 "No One"]
[Player2 "N/A"]
[Round "342"]
[Result "It Works!"]
[Size "5"]
[TPS "x5/x3,2112S,x/x5/x,1221,x3/x5 1 1"]

1. a3 c2
2. c2> {What a nub} a3+
3. d2+ a4>
4. d3- b4-
5. d2< Cc5? {Can you even believe this guy?}
6. c2+ b3>'
7. a5 2c3-2!
"""

    def test_parse_ptn(self):
        ptn = tak.ptn.PTN.parse(self.TEST_GAME)

        assert ptn.tags == {
            "Event": "PTN Viewer Demo",
            "Site": "Here",
            "Date": "2015.11.21",
            "Player1": "No One",
            "Player2": "N/A",
            "Round": "342",
            "Result": "It Works!",
            "Size": "5",
            "TPS": "x5/x3,2112S,x/x5/x,1221,x3/x5 1 1",
        }

        assert ptn.moves == [
            tak.ptn.parse_move(m)
            for m in [
                "a3",
                "c2",
                "c2>",
                "a3+",
                "d2+",
                "a4>",
                "d3-",
                "b4-",
                "d2<",
                "Cc5",
                "c2+",
                "b3>",
                "a5",
                "2c3-2",
            ]
        ]


class TestTPS(object):
    def test_parse_tps(self):
        tps = "x3,12,2S/x,22S,22C,11,21/121,212,12,1121C,1212S/21S,1,21,211S,12S/x,21S,2,x2 1 26"
        p = tak.ptn.parse_tps(tps)
        assert p.ply == 50
        assert p.size == 5

    def test_format_tps(self):
        tps = "x3,12,2S/x,22S,22C,11,21/121,212,12,1121C,1212S/21S,1,21,211S,12S/x,21S,2,x2 1 26"
        p = tak.ptn.parse_tps(tps)
        assert tak.ptn.format_tps(p) == tps
