from . import encoding
from attrs import define

import torch
from torch import nn
import typing as T


@define
class ModelWrapper:
    model: nn.Module
    device: T.Optional[str] = None

    def evaluate(self, pos):
        with torch.no_grad():
            encoded = torch.tensor(
                [encoding.encode(pos)], dtype=torch.long, device=self.device
            )
            out = self.model(encoded)
            return torch.softmax(out["moves"][0], dim=0).cpu(), out["values"][0].item()
