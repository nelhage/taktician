import json
import os.path
import subprocess
import sys
import tempfile

from tak import alphazero

import pytest
import yaml
import torch
import xformer.yaml_ext  # noqa

HERE = os.path.realpath(os.path.dirname(__file__))
SCRIPTS = os.path.realpath(os.path.join(HERE, "../scripts/"))


@pytest.mark.parametrize("wandb", [False, True])
def test_pipeline(wandb):
    if wandb and not os.environ.get("TEST_WANDB", "false").lower() == "true":
        pytest.skip("Skipping WANDB (slow); set TEST_WANDB=true to test.")

    with tempfile.TemporaryDirectory() as tmp:
        subprocess.check_call(
            [
                os.path.join(SCRIPTS, "encode_corpus"),
                "--analysis",
                "--test-frac=0.1",
                "--output",
                os.path.join(tmp, "corpus"),
                os.path.join(HERE, "corpus.csv"),
            ]
        )

        train = torch.load(os.path.join(tmp, "corpus-train.pt"))
        assert isinstance(train, dict)
        assert train["moves"].size(0) == 18

        subprocess.check_call(
            [
                sys.executable,
                os.path.join(SCRIPTS, "train_tak.py"),
                "--layers=1",
                "--d_model=64",
                "--device=cpu",
                "--data",
                os.path.join(tmp, "corpus"),
                "--steps=2",
                "--test-freq=2",
                "--save-freq=2",
                "--save-dir",
                os.path.join(tmp, "model"),
            ]
            + (["--wandb"] if wandb else []),
            cwd=tmp,
            env={"WANDB_MODE": "offline", **os.environ},
        )

        assert os.listdir(os.path.join(tmp, "model")) == ["step_000002"]
        save_dir = os.path.join(tmp, "model", "step_000002")
        with open(os.path.join(save_dir, "stats.json")) as fh:
            stats = json.load(fh)

        assert stats["step"] == 2
        assert "test_loss" in stats["metrics"]


def test_alphazero():
    with tempfile.TemporaryDirectory() as tmp:
        subprocess.check_call(
            [
                sys.executable,
                os.path.join(SCRIPTS, "alpha_zero.py"),
                "--layers=1",
                "--d_model=64",
                "--device=cpu",
                "--rollouts-per-step=5",
                "--rollout-simulations=5",
                "--rollout-workers=2",
                "--train-positions=128",
                "--batch=64",
                "--steps=2",
                "--no-progress",
                f"--save-dir={tmp}",
                f"--save-freq=1",
            ]
        )
        run_path = os.path.join(tmp, "run.yaml")
        with open(run_path, "r") as fh:
            config = yaml.unsafe_load(fh)
        assert isinstance(config, alphazero.Config)
        config.train_steps = 4
        config.save_freq = 5
        with open(run_path, "w") as fh:
            yaml.dump(config, fh)

        latest = os.path.join(tmp, "latest")
        assert os.path.exists(latest)
        assert os.path.islink(latest)
        assert os.readlink(latest) == "step_000002"
        subprocess.check_call(
            [
                sys.executable,
                os.path.join(SCRIPTS, "alpha_zero.py"),
                "--no-progress",
                f"--save-dir={tmp}",
                f"--load-model={tmp}/latest",
            ]
        )
        assert os.readlink(latest) == "step_000004"
