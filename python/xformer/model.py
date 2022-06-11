#!/usr/bin/env python
import torch
from torch import nn

import math

from dataclasses import dataclass
from functools import cached_property, lru_cache


class TextUnembedding(nn.Module):
    def __init__(self, cfg, dtype=None, device=None):
        super().__init__()

        self.final_ln = nn.LayerNorm(cfg.d_model, dtype=dtype, device=device)
        self.unembedding = nn.Linear(
            cfg.d_model, cfg.n_vocab, dtype=dtype, device=device
        )

    def init_weights(self, cfg):
        self.final_ln.reset_parameters()
        self.unembedding.weight.data.normal_(mean=0.0, std=cfg.initializer_range)

    def forward(self, acts):
        acts = self.final_ln(acts)
        return self.unembedding(acts)


@dataclass
class Config:
    n_vocab: int
    n_layer: int
    d_model: int
    d_head: int
    n_ctx: int = 1024
    initializer_range: float = 0.02
    positional_encoding: str = "sin"

    output_head: type = TextUnembedding

    autoregressive_mask: bool = True

    @cached_property
    def d_mlp(self):
        return 4 * self.d_model

    @cached_property
    def n_head(self):
        assert self.d_model % self.d_head == 0
        return self.d_model // self.d_head

    @cached_property
    def n_parameters(self):
        return self.n_layer * (2 * self.d_mlp * self.d_model + 4 * self.d_model**2)


class Resblock(nn.Module):
    def __init__(self, cfg: Config, dtype=None, device=None):
        super().__init__()
        self.attn_ln = nn.LayerNorm(cfg.d_model, dtype=dtype, device=device)
        self.attn = nn.MultiheadAttention(
            cfg.d_model, cfg.n_head, batch_first=True, dtype=dtype, device=device
        )

        self.mlp_ln = nn.LayerNorm(cfg.d_model, dtype=dtype, device=device)
        self.mlp_up = nn.Linear(cfg.d_model, cfg.d_mlp, dtype=dtype, device=device)
        self.mlp_act = nn.ReLU()
        self.mlp_down = nn.Linear(cfg.d_mlp, cfg.d_model, dtype=dtype, device=device)

        self.config = cfg

    def init_weights(self, cfg: Config):
        std = cfg.initializer_range / math.sqrt(cfg.n_layer)
        self.attn_ln.reset_parameters()
        self.attn.in_proj_weight.data.normal_(mean=0, std=std)
        self.attn.out_proj.weight.data.normal_(mean=0, std=std)
        self.mlp_ln.reset_parameters()
        self.mlp_up.weight.data.normal_(mean=0, std=std)
        self.mlp_down.weight.data.normal_(mean=0, std=std)

    def forward(self, resid):
        n_batch, n_ctx, d_model = resid.shape

        attn_ln = self.attn_ln(resid)

        attn_out, attn_pattern = self.attn(
            attn_ln,
            attn_ln,
            attn_ln,
            attn_mask=self.ar_mask(n_ctx, dtype=resid.dtype, device=resid.device)
            if self.config.autoregressive_mask
            else None,
        )
        resid = resid + attn_out

        mlp_ln = self.mlp_ln(resid)
        mlp_out = self.mlp_down(self.mlp_act(self.mlp_up(mlp_ln)))
        return resid + mlp_out

    @lru_cache
    def ar_mask(self, n_ctx, dtype, device):
        return torch.triu(
            torch.ones((n_ctx, n_ctx), dtype=torch.bool, device=device), diagonal=1
        )


class PositionalEncoding(nn.Module):
    def __init__(self, d_model: int, max_n_ctx: int = 2048, device=None, dtype=None):
        super().__init__()

        position = torch.arange(max_n_ctx, device=device, dtype=dtype).unsqueeze(1)
        div_term = torch.exp(
            torch.arange(0, d_model, 2, device=device, dtype=dtype)
            * (-math.log(10000.0) / d_model)
        )
        pe = torch.zeros((max_n_ctx, d_model), dtype=dtype, device=device)
        pe[:, 0::2] = torch.sin(position * div_term)
        pe[:, 1::2] = torch.cos(position * div_term)
        self.register_buffer("pe", pe)

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        """
        Args:
            x: Tensor, shape [batch_size, n_ctx, d_model]
        """
        x = x + self.pe[: x.size(1)]
        return x


class LearnedPositionalEncoding(nn.Module):
    def __init__(self, d_model: int, max_n_ctx: int = 2048, device=None, dtype=None):
        super().__init__()

        self.pe = nn.Parameter(
            torch.empty((max_n_ctx, d_model), dtype=dtype, device=device)
        )

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        """
        Args:
            x: Tensor, shape [batch_size, n_ctx, d_model]
        """
        x = x + self.pe[: x.size(1)]
        return x


class Torso(nn.Module):
    def __init__(self, cfg, dtype=None, device=None):
        super().__init__()

        self.layers = nn.ModuleList(
            [Resblock(cfg, dtype=dtype, device=device) for _ in range(cfg.n_layer)]
        )

    def forward(self, acts):
        for layer in self.layers:
            acts = layer(acts)
        return acts

    def init_weights(self, cfg):
        for layer in self.layers:
            layer.init_weights(cfg)


class TextEmbedding(nn.Module):
    def __init__(self, cfg, dtype=None, device=None):
        super().__init__()

        self.embedding = nn.Embedding(
            cfg.n_vocab, cfg.d_model, dtype=dtype, device=device
        )
        if cfg.positional_encoding == "sin":
            self.positional_encoding = PositionalEncoding(
                d_model=cfg.d_model, max_n_ctx=cfg.n_ctx, dtype=dtype, device=device
            )
        elif cfg.positional_encoding == "learned":
            self.positional_encoding = LearnedPositionalEncoding(
                d_model=cfg.d_model, max_n_ctx=cfg.n_ctx, dtype=dtype, device=device
            )
        elif cfg.positional_encoding == "none":
            self.positional_encoding = lambda acts: acts
        else:
            raise ValueError(
                f"Unknown positional encoding type: {cfg.positional_encoding!r}"
            )

    def init_weights(self, cfg):
        self.embedding.weight.data.normal_(
            mean=0.0, std=cfg.initializer_range * math.sqrt(cfg.d_model)
        )
        if isinstance(self.positional_encoding, LearnedPositionalEncoding):
            self.positional_encoding.pe.data.normal_(
                mean=0.0, std=cfg.initializer_range
            )

    def forward(self, tokens):
        acts = self.embedding(tokens)
        acts = self.positional_encoding(acts)
        return acts


class Transformer(nn.Module):
    def __init__(self, cfg, dtype=None, device=None):
        super().__init__()
        self.cfg = cfg

        self.embedding = TextEmbedding(cfg, dtype=dtype, device=device)
        self.torso = Torso(cfg, dtype=dtype, device=device)
        self.unembedding = cfg.output_head(cfg, dtype=dtype, device=device)

    def init_weights(self):
        self.embedding.init_weights(self.cfg)
        self.torso.init_weights(self.cfg)
        if hasattr(self.unembedding, "init_weights"):
            self.unembedding.init_weights(self.cfg)

    def forward(self, input):
        acts = self.embedding(input)
        acts = self.torso(acts)
        return self.unembedding(acts)
