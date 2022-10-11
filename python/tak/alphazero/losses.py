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
        probs = torch.softmax(m_logits, -1)
        accuracy = (probs * moves).sum(-1).mean()

        argmax = m_logits.argmax(-1, keepdims=True)
        top1_acc = torch.gather(moves, -1, argmax).mean()

        metrics = {
            "v_error": v_error.item(),
            "accuracy": accuracy.item(),
            "acc@01": top1_acc.item(),
        }

        return (v_error - accuracy), metrics
