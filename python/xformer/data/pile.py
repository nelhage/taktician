import zstandard as zstd
import json
import torch
import io


def pile_iterator(path):
    with open(path, "rb") as fh:
        dctx = zstd.ZstdDecompressor()
        with dctx.stream_reader(fh, read_size=8192) as reader:
            text_stream = io.TextIOWrapper(reader, encoding="utf-8")
            for line in text_stream:
                yield json.loads(line)


class PileDataset(torch.utils.data.IterableDataset):
    def __init__(self, path, n_ctx):
        self.path = path
        self.n_ctx = n_ctx

    def __iter__(self):
        for data in pile_iterator(self.path):
            bs = bytearray(data["text"].encode("utf-8")[: self.n_ctx])
            yield torch.tensor(bs, dtype=torch.uint8)

    @staticmethod
    def collate(tensors):
        n_ctx = max(t.shape[0] for t in tensors)
        data = torch.zeros((len(tensors), n_ctx), dtype=torch.long)
        for i, t in enumerate(tensors):
            data[i, : len(t)] = t
        return data
