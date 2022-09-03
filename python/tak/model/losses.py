from attrs import define, field
import torch
from torch.nn import functional as F


class MaskedAR:
    def __init__(self):
        self.xent = nn.CrossEntropyLoss(reduction="none")

    def train_and_metrics(self, batch, logits):
        return (
            (self.xent(logits.permute(0, 2, 1), batch.targets) * batch.mask).mean(),
            {},
        )


@define(slots=False)
class PolicyValue:
    v_weight: float = 1.0
    policy_weight: float = 1.0

    def loss_and_metrics(self, batch, logits):
        v_logits = logits["values"]
        m_logits = logits["moves"]

        v_error = F.mse_loss(v_logits, batch.values)

        metrics = {
            "v_error": v_error.item(),
        }

        moves = batch.moves
        if moves.ndim == 1:
            with torch.no_grad():
                argmax = torch.argmax(m_logits, dim=-1)
                match = argmax == moves
                metrics["acc@01"] = match.float().mean().item()

        return (
            self.v_weight * v_error
            + self.policy_weight * F.cross_entropy(m_logits, moves)
        ), metrics
