import typing as T

import attrs
import torch
from attrs import define, field

from xformer import train


class BatchProtocol(train.Batch, T.Protocol):
    def __init__(self, data: dict[str, torch.Tensor]):
        ...


@define
class Batch(BatchProtocol):
    data: dict[str, torch.Tensor]

    @property
    def inputs(self):
        return self.data["inputs"]


def transient(**kwargs):
    kwargs.setdefault("metadata", {})["transient"] = True
    kwargs["init"] = False
    return field(**kwargs)


@define(getstate_setstate=False)
class Dataset:
    path: str
    batch_size: int
    batches: T.Optional[int] = None
    device: str = "cpu"
    seed: int = 0x12345678
    batch_class: type = Batch

    data: dict[str, torch.Tensor] = transient()
    generator: torch.Generator = transient()

    def __getstate__(self):
        return {
            f.name: getattr(self, f.name)
            for f in attrs.fields(type(self))
            if not f.metadata.get("transient")
        }

    def __setstate__(self, state):
        for (k, v) in state.items():
            setattr(self, k, v)
        self.__attrs_post_init__()

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

    def _next_epoch(self):
        perm = torch.randperm(len(self), generator=self.generator)
        return {k: self.pin(v[perm]) for (k, v) in self.data.items()}

    def fastforward_epochs(self, n: int):
        for _ in range(n):
            self._next_epoch()

    def __iter__(self):
        shuffled = self._next_epoch()
        for i in range(0, len(self), self.batch_size):
            yield self.batch_class(
                {
                    k: v[i : i + self.batch_size].to(self.device)
                    for (k, v) in shuffled.items()
                }
            )


@define
class EpochIterator:
    shuffled: dict[str, torch.Tensor]
    batch_class: type
    i: int = 0

    def __next__(self):
        pass
