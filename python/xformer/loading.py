import os.path

import torch
import yaml

from .model import Transformer


def load_model(save_dir, device="cpu"):
    with open(os.path.join(save_dir, "config.yaml")) as fh:
        config = yaml.unsafe_load(fh)
    state = torch.load(os.path.join(save_dir, "model.pt"), map_location=device)
    model = Transformer(config, device=device)
    model.load_state_dict(state)
    return model


def save_model(model: Transformer, save_dir: str):
    os.makedirs(save_dir, exist_ok=True)

    torch.save(
        model.state_dict(),
        os.path.join(save_dir, "model.pt"),
    )
    with open(os.path.join(save_dir, "config.yaml"), "w") as fh:
        yaml.dump(model.cfg, fh)
