from xformer import data
from xformer.data import record_file

import pytest
import os.path
import tempfile

def test_pile():
  ds = data.PileDataset(os.path.join(os.path.dirname(__file__), '../data/pile/train/00.jsonl.zst'), n_ctx=1024)
  it = iter(ds)
  for _ in range(10):
    ex = next(it)
    assert ex.ndim == 1
    assert ex.shape[0] <= 1024

def test_record_file():
  with tempfile.TemporaryFile() as fh:
    w = os.fdopen(os.dup(fh.fileno()), 'w+b')

    with record_file.Writer(w) as wf:
      wf.write(b'hello, world')
      wf.write(b'goodnight, moon')

    fh.seek(0)

    with record_file.Reader(fh) as rf:
      it = iter(rf)
      assert next(it) == b'hello, world'
      assert next(it) == b'goodnight, moon'
      with pytest.raises(StopIteration):
        next(it)
