from .trainer import Hook, TrainState

import attrs
from attrs import define, field

from functools import partial
import secrets
import time
import wandb

from typing import Optional


class TimingHook(Hook):
    def before_rollout(self, state: TrainState):
        state.step_start = time.monotonic()

    def before_train(self, state: TrainState):
        state.step_stats["rollout_time"] = time.monotonic() - state.step_start
        state.train_start = time.monotonic()

    def after_step(self, state: TrainState):
        now = time.monotonic()
        state.step_stats["train_time"] = now - state.train_start
        state.step_stats["step_time"] = now - state.step_start


@define
class WandB(Hook):
    job_name: Optional[str] = None
    job_id: str = field(factory=partial(secrets.token_hex, 8))
    project: str = "taktician-alphazero"

    def before_run(self, state: TrainState, config):
        state.wandb = wandb.init(
            project=self.project,
            name=self.job_name,
            id=self.job_id,
            resume="allow",
        )
        wandb.config.update(attrs.asdict(config), allow_val_change=True)

    def finalize(self, state: TrainState):
        state.wandb.log(
            {
                "train_epoch": state.elapsed.epoch,
                "positions": state.elapsed.positions,
            }
            | state.step_stats
        )
