from attrs import define, field
from tak import mcts
import torch
from typing import Optional


@define(slots=False)
class Config:
    device: str = "cuda"
    server_port: int = 5432

    lr: float = 1e-3

    size: int = 3

    rollout_config: mcts.Config = field(
        factory=lambda: mcts.Config(
            simulation_limit=25,
            root_noise_alpha=1.0,
            root_noise_mix=0.25,
        )
    )

    rollout_resignation_threshold: float = 0.95
    rollout_ply_limit: int = 100

    rollout_workers: int = 50
    rollouts_per_step: int = 100
    replay_buffer_steps: int = 4

    train_batch: int = 64
    train_positions: int = 1024

    train_dtype: torch.dtype = torch.float32
    serve_dtype: torch.dtype = torch.float16

    save_path: Optional[str] = None
    save_freq: int = 10

    train_steps: int = 10

    wandb: bool = False
    job_name: Optional[str] = None
    project: str = "taktician-alphazero"

    def __attrs_post_init__(self):
        if self.device == "cpu":
            self.serve_dtype = torch.float32
