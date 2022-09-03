import torch
from torch import nn
from torch.nn import functional as F
from functools import cached_property


class ReferenceAccuracy:
    def loss_and_metrics(self, batch, logits):
        v_logits = logits["values"]
        m_logits = logits["moves"]

        v_error = F.mse_loss(v_logits, batch.values)

        moves = batch.moves
        moves = moves.to(m_logits.dtype)
        probs = (torch.softmax(m_logits, -1) * moves).sum(-1)
        accuracy = probs.mean()

        metrics = {
            "v_error": v_error.item(),
            "accuracy": accuracy.item(),
        }

        return (v_error - accuracy), metrics
