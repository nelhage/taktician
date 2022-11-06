import json
import os.path

import attrs
import torch
import yaml
from attrs import define

from xformer import loading
from .. import run


@define
class Save(run.Hook):
    save_dir: str
    step_freq: int

    def after_step(self, run, stats):
        if stats.step % self.step_freq != 0:
            return
        self.save_run(run, stats)

    def after_run(self, run, stats):
        self.save_run(run, stats)

    def save_run(self, run: run.Run, stats: run.Stats):
        run_dir = os.path.join(self.save_dir, f"step_{stats.step:06d}")
        print(f"Saving to {run_dir}...")
        loading.save_model(run.model, run_dir)
        with open(os.path.join(run_dir, "stats.json"), "w") as fh:
            json.dump(attrs.asdict(stats), fh, indent=2)
        # torch.save(os.path.join(run_dir, "model.opt.pt"), run.optimizer.state_dict())
