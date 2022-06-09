from xformer import data
from xformer.data import record_file
from xformer.scripts import split_pile

import pytest
import os.path
import tempfile

from dataclasses import dataclass
from typing import List, Tuple


@pytest.mark.skip("deprecated due to needing data")
def test_pile():
    ds = data.PileDataset(
        os.path.join(os.path.dirname(__file__), "../data/pile/train/00.jsonl.zst"),
        n_ctx=1024,
    )
    it = iter(ds)
    for _ in range(10):
        ex = next(it)
        assert ex.ndim == 1
        assert ex.shape[0] <= 1024


def test_record_file():
    with tempfile.TemporaryFile() as fh:
        w = os.fdopen(os.dup(fh.fileno()), "w+b")

        with record_file.Writer(w) as wf:
            wf.write(b"hello, world")
            wf.write(b"goodnight, moon")

        fh.seek(0)

        with record_file.Reader(fh) as rf:
            it = iter(rf)
            assert next(it) == b"hello, world"
            assert next(it) == b"goodnight, moon"
            with pytest.raises(StopIteration):
                next(it)


def test_join_batches():
    @dataclass
    class TestCase:
        texts: List[bytes]
        expect: List[Tuple[bytes, List[int]]]
        n_ctx: int = 8

    cases = [
        TestCase(
            texts=[b"abc", b"def"], expect=[(b"\0abc\0def", [1, 1, 1, 1, 2, 2, 2, 2])]
        ),
        TestCase(
            texts=[b"abc" * 4],
            expect=[
                (b"\0abcabca", [1] * 8),
                (b"\0bcabc\0\0", [1] * 6 + [0, 0]),
            ],
        ),
        TestCase(
            texts=[b"a" * 6, b"cc"],
            expect=[
                (b"\0aaaaaa\0", [1] * 7 + [0]),
                (b"\0cc\0\0\0\0\0", [1] * 3 + [0] * 5),
            ],
        ),
    ]

    for tc in cases:
        got = list(split_pile.join_batches(tc.texts, tc.n_ctx))
        assert len(got) == len(tc.expect)
        for (out, expect) in zip(got, tc.expect):
            assert out["text"].numpy().tobytes() == expect[0]
            assert out["text_nos"].tolist() == expect[1]
