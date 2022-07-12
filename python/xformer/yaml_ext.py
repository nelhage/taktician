import torch
import yaml


def dtype_representer(dumper, data):
    return dumper.represent_scalar("!dtype", data.__reduce__())


def dtype_constructor(loader, node):
    value = loader.construct_scalar(node)
    dtype = getattr(torch, value, None)
    if not isinstance(dtype, torch.dtype):
        raise ValueError(f"Invalid type: {value}")
    return dtype


yaml.add_representer(torch.dtype, dtype_representer)
yaml.add_constructor("!dtype", dtype_constructor)
