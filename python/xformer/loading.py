import os.path

import torch
import yaml

from .model import Transformer


def load_model(save_dir, map_location="cpu"):
    with open(os.path.join(save_dir, "config.yaml")) as fh:
        config = yaml.unsafe_load(fh)
    state = torch.load(os.path.join(save_dir, "model.pt"), map_location=map_location)
    model = Transformer(config)
    model.load_state_dict(state)
    return model
