from tak.model import encoding

import torch
from torch import nn

import typing as T  # noqa


class PolicyValue(nn.Module):
    def __init__(self, cfg, dtype=None, device=None):
        super().__init__()
        self.final_ln = nn.LayerNorm(
            normalized_shape=(cfg.d_model,), dtype=dtype, device=device
        )
        self.v_proj = nn.Linear(cfg.d_model, 1, dtype=dtype, device=device)
        self.move_proj = nn.Linear(
            cfg.d_model, 3 * encoding.MAX_SLIDES, dtype=dtype, device=device
        )

    def init_weights(self, cfg):
        pass

    def forward(self, acts):
        acts = self.final_ln(acts)[:, -1]

        v = torch.tanh(self.v_proj(acts))

        moves = self.move_proj(acts).reshape(-1, 3, encoding.MAX_SLIDES)

        return {
            "values": v.squeeze(-1),
            "moves": moves,
        }