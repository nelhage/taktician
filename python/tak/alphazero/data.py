from attrs import define, field
import torch


@define
class ReplayBufferBatch:
    data: dict[str, torch.Tensor]

    @property
    def inputs(self):
        return self.data["positions"]

    @property
    def mask(self):
        return self.data["mask"]

    @property
    def extra_inputs(self):
        return (~self.mask,)

    @property
    def moves(self):
        return self.data["moves"]

    @property
    def values(self):
        # Experiment with rollout result here instead
        return self.data["values"]


@define
class ReplayBufferDataset:
    replay_buffer: list[dict[str, torch.Tensor]]
    batch_size: int
    device: str
    flat_replay_buffer: dict[str, torch.Tensor] = field(init=False)

    def __attrs_post_init__(self):
        self.flat_replay_buffer = self.cat_replay_buffer()

    def cat_replay_buffer(self):
        full_replay_buffer = {
            k: torch.cat([d[k] for d in self.replay_buffer])
            for k in self.replay_buffer[0]
            if k not in ["positions", "mask"]
        }
        npos = sum(b["positions"].size(0) for b in self.replay_buffer)
        maxwidth = max(b["positions"].size(1) for b in self.replay_buffer)
        positions = torch.zeros((npos, maxwidth), dtype=torch.long)
        mask = torch.zeros((npos, maxwidth), dtype=torch.bool)

        n = 0
        for b in self.replay_buffer:
            shape = b["positions"].shape
            positions[n : n + shape[0], : shape[1]] = b["positions"]
            mask[n : n + shape[0], : shape[1]] = b["mask"]
            n += shape[0]

        full_replay_buffer["positions"] = positions
        full_replay_buffer["mask"] = mask
        return full_replay_buffer

    def pin(self, tensor):
        if self.device.startswith("cuda"):
            return tensor.pin_memory()
        return tensor

    def __iter__(self):
        npos = len(self.flat_replay_buffer["positions"])

        perm = torch.randperm(npos)
        shuffled = {k: self.pin(v[perm]) for (k, v) in self.flat_replay_buffer.items()}

        for i in range(0, npos, self.batch_size):
            yield ReplayBufferBatch(
                {
                    k: v[i : i + self.batch_size].to(self.device)
                    for (k, v) in shuffled.items()
                }
            )
