from tak.model import encoding
import tak.ptn


def test_encoding_invariants():
    assert len(encoding.Token.CAPSTONES) == encoding.MAX_CAPSTONES
    assert len(encoding.Token.RESERVES) == encoding.MAX_RESERVES

    assert set(encoding.Token.CAPSTONES) & set(encoding.Token.RESERVES) == set()


def test_encode():
    p1 = tak.ptn.parse_tps("12,x,22S/x2,1/x,21,1 2 8")
    p2 = tak.ptn.parse_tps("x,2,2/x,12S,1/x,1,2S 1 6")

    e1 = encoding.encode(p1)
    e2 = encoding.encode(p2)

    batch, mask = encoding.encode_batch([p1, p2])

    assert [b for (b, m) in zip(batch[0].tolist(), mask[0].tolist()) if m] == e1
    assert [b for (b, m) in zip(batch[1].tolist(), mask[1].tolist()) if m] == e2
