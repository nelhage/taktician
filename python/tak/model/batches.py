import torch
from torch.nn import functional as F

from attrs import define


@define
class Position:
    data: dict[str, torch.Tensor]

    @property
    def inputs(self):
        return self.data["positions"][:, :-1]

    @property
    def targets(self):
        return self.data["positions"][:, 1:]

    @property
    def mask(self):
        return self.data["mask"][:, :-1]


OUTPUT_SENTINEL = 256


@define
class PositionValuePolicy:
    data: dict[str, torch.Tensor]

    @property
    def inputs(self):
        return F.pad(self.data["positions"], (0, 1), value=OUTPUT_SENTINEL)

    @property
    def moves(self):
        return self.data["moves"]

    @property
    def moves_mask(self):
        return self.data["moves_mask"]

    @property
    def values(self):
        return self.data["values"]
