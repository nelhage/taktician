from collections import defaultdict

import torch
from attrs import define

from xformer.data import Dataset
from tak.alphazero import losses
from xformer.train import LossFunction
from ..trainer import Hook, TrainState
from functools import cached_property


@define(slots=False)
class TestLoss(Hook):
    __test__ = False

    dataset: Dataset
    frequency: int
    loss: LossFunction = losses.ReferenceAccuracy()

    name: str = "test"

    def after_step(self, state: TrainState):
        if state.elapsed.step > 1 and state.elapsed.step % self.frequency != 0:
            return

        with torch.no_grad():
            losses = []
            metrics = defaultdict(float)
            for batch in self.dataset:
                out = state.model(batch.inputs, *batch.extra_inputs)
                loss, batch_metrics = self.loss.loss_and_metrics(batch, out)
                losses.append(loss)
                for (k, v) in batch_metrics.items():
                    metrics[k] += v
            for (k, v) in metrics.items():
                metrics[k] = v / len(losses)
            test_loss = torch.stack(losses).mean().item()

        for (k, v) in metrics.items():
            state.step_stats[f"{self.name}.{k}"] = v
        state.step_stats[f"{self.name}.loss"] = test_loss
