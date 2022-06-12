import torch

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


@define
class PositionValuePolicy:
    data: dict[str, torch.Tensor]

    @property
    def inputs(self):
        return self.data["positions"]

    @property
    def moves(self):
        return self.data["moves"]

    @property
    def values(self):
        return self.data["values"]
