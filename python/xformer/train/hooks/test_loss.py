from attrs import define

from ..run import Dataset, Hook, Run, Stats

import torch


@define
class TestLoss(Hook):
    dataset: Dataset
    frequency: int

    def after_step(self, run: Run, stats: Stats):
        if stats.step > 1 and stats.step % self.frequency != 0:
            return

        with torch.no_grad():
            test_loss = (
                torch.stack(
                    [run.loss(batch, run.model(batch.inputs)) for batch in self.dataset]
                )
                .mean()
                .item()
            )
        print(f"[step={stats.step:06d}] test_loss={test_loss:4.2f}")
        stats.metrics["test_loss"] = test_loss
