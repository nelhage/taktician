import os.path
import typing as T
from functools import partial

from attrs import define, field
from torch.profiler import ProfilerAction, profile

from ..run import Hook, Run, Stats


@define
class Profile(Hook):
    extra_steps: set[int] = field(factory=set)
    every: T.Optional[int] = None
    output_root: str = "profile/"

    profiler: profile = field(init=False)

    def before_run(self, run: Run):
        run.profiler = profile(
            schedule=self.schedule,
            with_stack=True,
            on_trace_ready=partial(self.save_profile, run=run),
        )
        run.profiler.start()

    def should_profile(self, step):
        if step in self.extra_steps:
            return True
        if self.every is not None:
            return step % self.every == 0
        return False

    def schedule(self, step):
        if self.should_profile(step):
            print(f"Profiling step {step}...")
            return ProfilerAction.RECORD_AND_SAVE
        if self.should_profile(step + 1):
            return ProfilerAction.WARMUP
        return ProfilerAction.NONE

    def save_profile(self, prof, run):
        os.makedirs(self.output_root, 0o755, True)
        prof.export_chrome_trace(
            os.path.join(
                self.output_root, f"step_{run.profiler.step_num-1}.pt.trace.json"
            )
        )

    def after_step(self, run: Run, stats: Stats):
        run.profiler.step()
