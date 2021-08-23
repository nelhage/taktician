from xformer import data
import os.path

def test_pile():
  ds = data.PileDataset(os.path.join(os.path.dirname(__file__), '../data/pile/train/00.jsonl.zst'), n_ctx=1024)
  it = iter(ds)
  for _ in range(10):
    ex = next(it)
    assert ex.ndim == 1
    assert ex.shape[0] <= 1024
