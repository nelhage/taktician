import time
import torch

from . import run


class Trainer:
    start_time: float
    run: run.Run
    stats: run.Stats
    opt: torch.optim.Optimizer

    def __init__(self, training_run: run.Run):
        self.run = training_run
        self.stats = run.Stats()

    def train(self):
        self.start_time = time.time()
        self.epoch = iter(self.run.dataset)

        self.run.model.init_weights()

        for hook in self.run.hooks:
            hook.before_run(self.run)

        self.opt = torch.optim.AdamW(
            self.run.model.parameters(), lr=self.run.optimizer.lr
        )
        while True:
            self.one_step()
            if self.run.stop(self.stats):
                break

    def one_step(self):
        step_start = time.time()
        self.stats.step += 1
        self.stats.metrics.clear()

        for hook in self.run.hooks:
            hook.before_step(self.run, self.stats)

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

        for hook in self.run.hooks:
            hook.after_step(self.run, self.stats)

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


__all__ = ["Trainer"]
