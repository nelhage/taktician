import attrs
import wandb
import time
import torch

from . import run
from .. import model


class Trainer:
    start_time: float
    run: run.Run
    stats: run.Stats
    opt: torch.optim.Optimizer

    def __init__(self, training_run: run.Run):
        self.run = training_run
        self.stats = run.Stats()

    def config_to_log(self):
        return self.run.logging.config

    def train(self):
        self.start_time = time.time()
        self.epoch = iter(self.run.dataset)

        if self.run.logging.wandb:
            job_name = self.run.logging.job_name
            if job_name is not None and "{rand}" in job_name:
                job_name = job_name.format(rand=wandb.util.generate_id())
            wandb.init(
                project=self.run.logging.project,
                name=job_name,
                group=self.run.logging.group,
            )
            wandb.config.update(self.config_to_log())

        self.run.model.init_weights()

        # TODO: profiler

        self.opt = torch.optim.AdamW(
            self.run.model.parameters(), lr=self.run.optimizer.lr
        )
        while True:
            self.one_step()
            if self.run.stop(self.stats):
                break

    def one_step(self):
        self.stats.step += 1

        step_start = time.time()
        self.opt.zero_grad(set_to_none=True)
        batch = next(self.epoch)

        inputs = batch.inputs
        self.stats.sequences += inputs.size(0)
        self.stats.tokens += inputs.numel()

        logits = self.run.model(inputs)
        loss = self.run.loss(batch, logits)
        self.stats.train_loss = loss.item()
        loss.backward()
        self.opt.step()

        # self.profiler.step()
        step_done = time.time()
        self.stats.step_time = step_done - step_start
        self.stats.elapsed_time = step_done - self.start_time

        self.log_step()

    def log_step(self):
        stats = self.stats
        print(
            f"[step={stats.step:06d}"
            f" t={stats.elapsed_time:.1f}s"
            f" sequences={stats.sequences:08d}]"
            f" loss={stats.train_loss:2.2f}"
            f" ms_per_step={1000*(stats.step_time):.0f}"
        )

        if self.run.logging.wandb:
            wandb.log(self.stats.__dict__, step=self.stats.step)


__all__ = ["Trainer"]
