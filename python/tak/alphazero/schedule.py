from .trainer import Scheduler, TrainState
from attrs import define


@define(slots=False)
class LinearWarmup:
    final_value: float
    warmup_steps: int = 100

    def value(self, state: TrainState) -> float:
        if state.elapsed.step >= self.warmup_steps:
            scale = 1.0
        else:
            scale = state.elapsed.step / self.warmup_steps
        return scale * self.final_value
