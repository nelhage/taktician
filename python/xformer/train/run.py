from attrs import define, field
import torch
from torch import nn

import typing as T


class Batch(T.Protocol):
    inputs: torch.Tensor


Dataset = T.Iterable[Batch]


class LossFunction(T.Protocol):
    def __call__(self, batch, output):
        ...

    def metrics(self, batch, output) -> dict[str, float]:
        return {}


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
    lr_schedule: T.Optional[T.Callable[[Stats], float]] = None


Trigger = T.Callable[[Stats], bool]


@define
class StopTrigger:
    steps: T.Optional[int]
    sequences: T.Optional[int]

    def __call__(self, stats: Stats):
        if self.steps is not None and stats.step >= self.steps:
            return True
        if self.sequences is not None and stats.sequences >= self.sequences:
            return True
        return False


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

    hooks: list[Hook] = field(factory=list)


__all__ = [
    "Batch",
    "Dataset",
    "Hook",
    "LossFunction",
    "Optimizer",
    "Run",
    "Stats",
    "StopTrigger",
]
