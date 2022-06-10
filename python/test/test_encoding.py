from tak.model import encoding


def test_encoding_invariants():
    assert len(encoding.Token.CAPSTONES) == encoding.MAX_CAPSTONES
    assert len(encoding.Token.RESERVES) == encoding.MAX_RESERVES

    assert set(encoding.Token.CAPSTONES) & set(encoding.Token.RESERVES) == set()
