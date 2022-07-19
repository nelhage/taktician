import os.path

import torch
from torch import nn
import yaml

from .model import Transformer


def load_config(save_dir):
    with open(os.path.join(save_dir, "config.yaml")) as fh:
        return yaml.unsafe_load(fh)


def load_snapshot(model: nn.Module, save_dir: str):
    state = torch.load(os.path.join(save_dir, "model.pt"), map_location="cpu")
    model.load_state_dict(state)


def load_model(save_dir, device="cpu"):
    config = load_config(save_dir)
    model = Transformer(config, device=device)
    load_snapshot(model, save_dir)
    return model


def save_model(model: Transformer, save_dir: str):
    os.makedirs(save_dir, exist_ok=True)

    torch.save(
        model.state_dict(),
        os.path.join(save_dir, "model.pt"),
    )
    with open(os.path.join(save_dir, "config.yaml"), "w") as fh:
        yaml.dump(model.cfg, fh)
