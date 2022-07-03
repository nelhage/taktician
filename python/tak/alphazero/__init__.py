from attrs import define, field


@define
class Config:
    device: str = "cuda"
    server_port: int = 5001

    size: int = 3

    rollout_workers: int = 50
    rollout_simulations: int = 25

    rollouts_per_step: int = 100
    replay_buffer_steps: int = 4

    train_batch: int = 64
    train_positions: int = 1024
