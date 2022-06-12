from attrs import define
from collections import defaultdict

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
            losses = []
            metrics = defaultdict(float)
            for batch in self.dataset:
                out = run.model(batch.inputs, *batch.extra_inputs)
                loss, batch_metrics = run.loss.loss_and_metrics(batch, out)
                losses.append(loss)
                for (k, v) in batch_metrics.items():
                    metrics[k] += v
            for (k, v) in metrics.items():
                metrics[k] = v / len(losses)
            test_loss = torch.stack(losses).mean().item()

        for (k, v) in metrics.items():
            stats.metrics[f"test.{k}"] = v
        stats.metrics["test_loss"] = test_loss
