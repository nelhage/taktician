from xformer import train

from attrs import define, field

import torch

import typing as T


class Batch(train.Batch, T.Protocol):
    def __init__(self, data: dict[str, torch.Tensor]):
        ...

    @property
    def inputs(self):
        return self.data["inputs"]


@define
class Dataset:
    path: str
    batch_size: int
    batches: T.Optional[int] = None
    device: str = "cpu"
    seed: int = 0x12345678
    batch_class: type = Batch

    data: dict[str, torch.Tensor] = field(init=False)
    generator: torch.Generator = field(init=False)

    def __attrs_post_init__(self):
        self.data = torch.load(self.path)
        for (k, v) in self.data.items():
            if v.dtype == torch.uint8:
                v = v.long()
            if self.batches is not None:
                v = v[: self.batches * self.batch_size]
            self.data[k] = v

        self.generator = torch.Generator().manual_seed(self.seed)

    def __len__(self):
        return len(next(iter(self.data.values())))

    def pin(self, tensor):
        if self.device.startswith("cuda"):
            return tensor.pin_memory()
        return tensor

    def __iter__(self):
        perm = torch.randperm(len(self), generator=self.generator)
        shuffled = {k: self.pin(v[perm]) for (k, v) in self.data.items()}
        for i in range(0, len(self), self.batch_size):
            yield self.batch_class(
                {
                    k: v[i : i + self.batch_size].to(self.device)
                    for (k, v) in shuffled.items()
                }
            )
