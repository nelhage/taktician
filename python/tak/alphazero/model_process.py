import typing as T  # noqa
import time
import os
import itertools

import torch
import queue
from torch import multiprocessing

import grpc
import asyncio

import wandb

import xformer
import yaml

from tak.proto import analysis_pb2_grpc
import tak.model.server
from tak.model import batches, losses

from attrs import field, define
import attrs

from .. import Config
from . import data


@define(kw_only=True)
class Command:
    id: int = field(factory=itertools.count().__next__)


class Shutdown(Command):
    pass


class WaitForStartup(Command):
    pass


@define
class TrainStep(Command):
    batch: dict[str, torch.Tensor]


@define
class SaveSnapshot(Command):
    snapshot_path: str


@define
class ModelServerShared:
    cmd: multiprocessing.Queue = field(factory=multiprocessing.Queue, init=False)
    reply: multiprocessing.Queue = field(factory=multiprocessing.Queue, init=False)


@define
class ModelServerHandle:
    model: xformer.Transformer
    config: Config
    process: multiprocessing.Process = field(init=False)
    shared: ModelServerShared = field(factory=ModelServerShared, init=False)

    def __attrs_post_init__(self):
        self.process = multiprocessing.Process(
            target=self._run_in_spawn, name="analysis_server"
        )

    def _run_in_spawn(self):
        print(f"Starting model process pid={os.getpid()}")
        worker = ModelServerProcess(
            model=self.model,
            config=self.config,
            shared=self.shared,
        )
        worker.run()

    def send(self, cmd: Command):
        self.shared.cmd.put(cmd)
        while True:
            try:
                got = self.shared.reply.get(timeout=1)
            except queue.Empty:
                if not self.process.is_alive():
                    raise RuntimeError(f"Child died unexpected!")
            else:
                break
        assert got == cmd.id, "Got a reply to the wrong command!"

    def start(self):
        self.process.start()
        self.send(WaitForStartup())

    def stop(self):
        self.shared.cmd.put(Shutdown())
        self.process.join()

    def train_step(self, batch):
        self.send(TrainStep(batch=batch))

    def save_model(self, path):
        self.send(SaveSnapshot(snapshot_path=path))


def create_server(
    model: xformer.Transformer,
    config: Config,
) -> ModelServerHandle:
    return ModelServerHandle(
        model=model,
        config=config,
    )


@define
class ModelServerProcess:
    model: xformer.Transformer
    config: Config
    shared: ModelServerShared

    wandb: T.Optional["wandb.Run"] = field(default=None, init=False)

    ready: asyncio.Event = field(factory=asyncio.Event)

    loop: asyncio.BaseEventLoop = field(init=False)
    server: grpc.aio.Server = field(init=False)
    tasks: list[asyncio.Task] = field(init=False, factory=list)
    replay_buffer: list[dict[str, torch.Tensor]] = field(init=False, factory=list)
    train_params: dict[str, torch.Tensor] = field(init=False)
    opt: torch.optim.AdamW = field(init=False)

    step: int = field(default=0, init=False)
    train_positions: int = field(default=0, init=False)
    train_epoch: int = field(default=0, init=False)
    last_step: float = field(init=False)

    def run(self):
        asyncio.run(self.run_async())

    def serve_mode(self):
        self.train_params = {k: v.cpu() for (k, v) in self.model.state_dict().items()}
        self.model.to(device=self.config.device, dtype=self.config.serve_dtype)

    def train_mode(self):
        self.model.to(self.config.train_dtype).load_state_dict(self.train_params)

    def train_step(self, batch):
        now = time.monotonic()
        rollout_time = now - self.last_step
        self.last_step = now

        self.replay_buffer.append(batch)
        if len(self.replay_buffer) > self.config.replay_buffer_steps:
            self.replay_buffer = self.replay_buffer[1:]

        self.train_mode()

        self.step += 1

        loss_fn = losses.PolicyValue()
        ds = data.ReplayBufferDataset(
            replay_buffer=self.replay_buffer,
            batch_size=self.config.train_batch,
            device=self.config.device,
        )

        unique = len(
            set(
                tuple(p[m].tolist())
                for (p, m) in zip(batch["positions"], batch["mask"])
            )
        )
        plies = len(batch["positions"])
        stats = {
            "rollout_plies": plies,
            "replay_buffer_plies": len(ds.flat_replay_buffer["positions"]),
            "rollout_unique_plies": unique,
            "train_step": self.step,
            "rollout_time": rollout_time,
        }

        self.train_epoch += 1
        train_start = time.monotonic()

        it = iter(ds)
        for i in range(0, self.config.train_positions, self.config.train_batch):
            try:
                self.train_epoch += 1
                batch = next(it)
            except StopIteration:
                it = iter(ds)
                batch = next(it)

            self.opt.zero_grad()
            out = self.model(batch.inputs, *batch.extra_inputs)
            loss, metrics = loss_fn.loss_and_metrics(batch, out)
            loss.backward()
            self.opt.step()

            self.train_positions += batch.size(0)

            if self.wandb is not None:
                self.wandb.log(
                    {
                        "train_loss": loss.item(),
                        "train_epoch": self.train_epoch,
                        "positions": self.train_positions,
                    }
                    | stats
                    | metrics
                )

        train_time = time.monotonic() - train_start
        print(
            f"step={self.step}"
            f" games={self.config.rollouts_per_step}"
            f" plies={plies}"
            f" unique={unique}"
            f" rollout_time={rollout_time:0.2f}s"
            f" train_time={train_time:0.2f}s"
            f" ply/s={plies/(rollout_time):.1f}s"
            f" last_loss={loss.item():0.2f}"
        )

        self.serve_mode()

    async def command_loop(self):
        while True:
            self.last_step = time.monotonic()
            event = await self.loop.run_in_executor(None, self.shared.cmd.get)
            if isinstance(event, WaitForStartup):
                await self.ready.wait()
            elif isinstance(event, Shutdown):
                await self.server.stop(2)
                return
            elif isinstance(event, TrainStep):
                self.train_step(event.batch)
            elif isinstance(event, SaveSnapshot):
                os.makedirs(event.snapshot_path, exist_ok=True)
                torch.save(
                    self.train_params,
                    os.path.join(event.snapshot_path, "model.pt"),
                )
                with open(os.path.join(event.snapshot_path, "config.yaml"), "w") as fh:
                    yaml.dump(self.model.cfg, fh)

            else:
                raise AssertionError(f"Unknown command: {event}")
            await self.loop.run_in_executor(None, self.shared.reply.put, event.id)

    async def run_async(self):
        if self.config.wandb:
            self.wandb = wandb.init(
                project=self.config.project, name=self.config.job_name
            )
            wandb.config.update(attrs.asdict(self.config))
        self.loop = asyncio.get_event_loop()

        self.opt = torch.optim.AdamW(self.model.parameters(), lr=self.config.lr)

        self.serve_mode()

        self.server = grpc.aio.server()
        self.server.add_insecure_port(f"localhost:{self.config.server_port}")

        analysis = tak.model.server.Server(model=self.model, device=self.config.device)

        self.tasks.append(asyncio.create_task(analysis.worker_loop()))
        self.tasks.append(asyncio.create_task(self.command_loop()))

        analysis_pb2_grpc.add_AnalysisServicer_to_server(
            analysis,
            self.server,
        )
        await self.server.start()
        self.ready.set()
        try:
            done, pending = await asyncio.wait(
                self.tasks + [self.server.wait_for_termination()],
                return_when=asyncio.FIRST_COMPLETED,
            )
            for task in pending:
                task.cancel()
            for task in done:
                task.result()
        finally:
            await self.server.stop(None)
