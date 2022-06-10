from attrs import define
import attrs

from ..run import Hook, Run, Stats

import typing as T

import torch


@define
class Wandb(Hook):
    job_name: T.Optional[str] = None
    project: T.Optional[str] = None
    group: T.Optional[str] = None
    config: T.Any = None

    def before_run(self, run: Run):
        import wandb

        job_name = self.job_name
        if job_name is not None and "{rand}" in job_name:
            job_name = job_name.format(rand=wandb.util.generate_id())
        run.wandb = wandb.init(
            project=self.project,
            name=job_name,
            group=self.group,
        )
        wandb.config.update(self.config)

    def after_step(self, run: Run, stats: Stats):
        run.wandb.log(attrs.asdict(stats), step=stats.step)
