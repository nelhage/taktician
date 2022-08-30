from ..trainer import Hook, TrainState

from attrs import define, field
import os.path

from xformer import loading
import torch
import yaml


def save_snapshot(state: TrainState, snapshot_path):
    os.makedirs(snapshot_path, exist_ok=True)
    loading.save_model(state.model, snapshot_path)
    torch.save(
        state.opt.state_dict(),
        os.path.join(snapshot_path, "opt.pt"),
    )
    torch.save(
        state.replay_buffer,
        os.path.join(snapshot_path, "replay_buffer.pt"),
    )
    with open(os.path.join(snapshot_path, "elapsed.yaml"), "w") as fh:
        yaml.dump(state.elapsed, fh)


@define(slots=False)
class SavingHook(Hook):
    freq: int

    def before_run(self, state, config):
        self.run_dir = config.run_dir

    def check_and_clear_save_request(self) -> bool:
        run_dir = self.run_dir
        if not run_dir:
            return False
        flagpath = os.path.join(run_dir, "SAVE_NOW")
        if os.path.exists(flagpath):
            os.unlink(flagpath)
            return True
        return False

    def after_step(self, state: TrainState):
        if state.elapsed.step % self.freq == 0 or self.check_and_clear_save_request():
            self.save_snapshot(state)

    def after_run(self, state: TrainState):
        self.save_snapshot(state)

    def save_snapshot(self, state: TrainState):
        save_dir = os.path.join(self.run_dir, f"step_{state.elapsed.step:06d}")
        print(f"Saving snapshot to {save_dir}...")
        save_snapshot(state, save_dir)
        latest_link = os.path.join(self.run_dir, "latest")
        try:
            os.unlink(latest_link)
        except FileNotFoundError:
            pass
        os.symlink(
            os.path.basename(save_dir),
            latest_link,
        )
