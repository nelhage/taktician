from attrs import define

from . import run


@define
class LinearWarmupCooldown:
    warmup_steps: int
    cooldown_steps: int
    cooldown_start: int

    def __call__(self, stats: run.Stats):
        if stats.step < self.warmup_steps:
            return stats.step / self.warmup_steps
        if stats.step > self.cooldown_start:
            end = self.cooldown_start + self.cooldown_steps
            remaining = end - stats.step
            return (remaining + 1) / self.cooldown_steps
        return 1.0
