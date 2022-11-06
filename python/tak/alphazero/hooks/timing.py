from ..trainer import Hook, TrainState
import time


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
