from . import encoding
from attrs import define, field

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


@define
class GraphedWrapper:
    model: nn.Module
    max_length: int = 30

    graph: torch.cuda.CUDAGraph = field(factory=torch.cuda.CUDAGraph)
    static_pos: torch.Tensor = field(init=False)
    static_mask: torch.Tensor = field(init=False)
    static_output: dict[str, torch.Tensor] = field(init=False)

    def __attrs_post_init__(self):
        self.static_pos = torch.ones(
            (
                1,
                self.max_length,
            ),
            dtype=torch.long,
            device="cuda",
        )
        self.static_mask = torch.zeros(
            (
                1,
                self.max_length,
            ),
            dtype=torch.bool,
            device="cuda",
        )

        s = torch.cuda.Stream()
        s.wait_stream(torch.cuda.current_stream())
        with torch.cuda.stream(s), torch.no_grad():
            for _ in range(3):
                self.model(self.static_pos, self.static_mask)
        torch.cuda.current_stream().wait_stream(s)

        with torch.cuda.graph(self.graph), torch.no_grad():
            self.static_output = self.model(self.static_pos, self.static_mask)

    def evaluate(self, pos):
        with torch.no_grad():
            encoded = encoding.encode(pos)
            self.static_pos[:, : len(encoded)].copy_(
                torch.tensor(encoded, dtype=torch.long)
            )
            self.static_mask[:, : len(encoded)].fill_(0)
            self.static_mask[:, len(encoded) :].fill_(0)
            self.graph.replay()
            out = self.static_output
            return torch.softmax(out["moves"][0], dim=0).cpu(), out["values"][0].item()
