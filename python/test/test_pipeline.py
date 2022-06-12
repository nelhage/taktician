import tempfile
import torch
import json

import sys
import subprocess
import os.path

HERE = os.path.realpath(os.path.dirname(__file__))
SCRIPTS = os.path.realpath(os.path.join(HERE, "../scripts/"))


def test_pipeline():
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
                "--wandb",
                "--data",
                os.path.join(tmp, "corpus"),
                "--steps=2",
                "--test-freq=2",
                "--save-freq=2",
                "--save-dir",
                os.path.join(tmp, "model"),
            ],
            cwd=tmp,
            env={"WANDB_MODE": "offline", **os.environ},
        )

        assert os.listdir(os.path.join(tmp, "model")) == ["step_000002"]
        save_dir = os.path.join(tmp, "model", "step_000002")
        with open(os.path.join(save_dir, "stats.json")) as fh:
            stats = json.load(fh)

        assert stats["step"] == 2
        assert "test_loss" in stats["metrics"]
