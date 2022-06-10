from attrs import define, field
import torch
from torch import nn

import typing as T


class Batch(T.Protocol):
    inputs: torch.Tensor


Dataset = T.Iterable[Batch]


class LossFunction:
    def compute(self, batch, output):
        pass


@define
class Profile:
    extra_steps: set[int]
    every: int
    output_root: str = "profile/"


@define
class Stats:
    step: int = 0
    sequences: int = 0
    tokens: int = 0
    train_loss: float = 0
    step_time: float = 0
    elapsed_time: float = 0
    metrics: dict[str, object] = field(factory=dict)


@define
class Optimizer:
    lr: float = 5e-4


Trigger = T.Callable[[Stats], bool]


class Hook:
    def before_run(self, run: "Run"):
        pass

    def before_step(self, run: "Run", stats: Stats):
        pass

    def after_step(self, run: "Run", stats: Stats):
        pass


@define(slots=False)
class Run:
    model: nn.Module
    dataset: Dataset
    loss: LossFunction

    stop: Trigger

    optimizer: Optimizer = field(factory=Optimizer)
    profile: T.Optional[Profile] = None

    hooks: list[Hook] = field(factory=list)


__all__ = [
    "Batch",
    "Dataset",
    "Hook",
    "LossFunction",
    "Optimizer",
    "Profile",
    "Run",
    "Stats",
]
