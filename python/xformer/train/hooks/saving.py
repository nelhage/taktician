from .. import run
import os.path
import torch
from attrs import define
import attrs
import yaml
import json


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
        os.makedirs(run_dir, exist_ok=True)
        print(f"Saving to {run_dir}...")
        torch.save(
            run.model.state_dict(),
            os.path.join(run_dir, "model.pt"),
        )
        with open(os.path.join(run_dir, "config.yaml"), "w") as fh:
            yaml.dump(run.model.cfg, fh)
        with open(os.path.join(run_dir, "stats.json"), "w") as fh:
            json.dump(attrs.asdict(stats), fh, indent=2)
        # torch.save(os.path.join(run_dir, "model.opt.pt"), run.optimizer.state_dict())
