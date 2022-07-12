import torch
import yaml
import io
import xformer.yaml_ext  # noqa


def test_yaml_dtype():
    out = yaml.dump({"dtype": torch.float32})
    got = yaml.unsafe_load(io.StringIO(out))
    assert got["dtype"] == torch.float32
