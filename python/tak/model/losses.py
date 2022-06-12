import torch
from torch import nn
from torch.nn import functional as F


class MaskedAR:
    def __init__(self):
        self.xent = nn.CrossEntropyLoss(reduction="none")

    def train_and_metrics(self, batch, logits):
        return (
            (self.xent(logits.permute(0, 2, 1), batch.targets) * batch.mask).mean(),
            {},
        )


class PolicyValue:
    v_weight: float = 1.0
    policy_weight: float = 1.0

    def __init__(self):
        self.xent = nn.CrossEntropyLoss(reduction="mean")

    def loss_and_metrics(self, batch, logits):
        v_logits = logits["values"]
        m_logits = logits["moves"]

        with torch.no_grad():
            argmax = torch.max(m_logits, dim=-1).indices
            match = argmax == batch.moves

        v_error = F.mse_loss(v_logits, batch.values)

        return (
            self.v_weight * v_error
            + self.policy_weight * self.xent(m_logits, batch.moves)
        ), {
            "v_error": v_error.item(),
            "acc@1": match.float().mean().item(),
        }
