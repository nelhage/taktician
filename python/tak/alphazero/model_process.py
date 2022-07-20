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
from xformer import loading
import yaml

from tak.proto import analysis_pb2_grpc
import tak.model.server
from tak.model import batches, losses

from attrs import field, define
import attrs

from .. import Config
from . import data, stats


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
class Reply:
    id: int
    reply: T.Any


@define
class ModelServerShared:
    cmd: multiprocessing.Queue = field(factory=multiprocessing.Queue, init=False)
    reply: multiprocessing.Queue = field(factory=multiprocessing.Queue, init=False)


@define
class ModelServerHandle:
    config: Config
    model_config: xformer.Config
    process: multiprocessing.Process = field(init=False)
    shared: ModelServerShared = field(factory=ModelServerShared, init=False)

    def __attrs_post_init__(self):
        self.process = multiprocessing.Process(
            target=self._run_in_spawn, name="analysis_server"
        )

    def _run_in_spawn(self):
        print(f"Starting model process pid={os.getpid()}")
        worker = ModelServerProcess(
            model=xformer.Transformer(
                self.model_config,
                device=self.config.device,
            ),
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
        assert got.id == cmd.id, "Got a reply to the wrong command!"
        return got.reply

    def start(self):
        self.process.start()
        self.send(WaitForStartup())

    def stop(self):
        self.shared.cmd.put(Shutdown())
        self.process.join()

    def train_step(self, batch):
        return self.send(TrainStep(batch=batch))


def create_server(
    config: Config,
    model_config: xformer.Config,
) -> ModelServerHandle:
    return ModelServerHandle(
        config=config,
        model_config=model_config,
    )


@define
class ModelServerProcess:
    model: xformer.Transformer
    config: Config
    shared: ModelServerShared

    wandb: T.Optional["wandb.Run"] = field(default=None, init=False)

    ready: asyncio.Event = field(factory=asyncio.Event, init=False)

    loop: asyncio.BaseEventLoop = field(init=False)
    server: grpc.aio.Server = field(init=False)
    tasks: list[asyncio.Task] = field(init=False, factory=list)
    replay_buffer: list[dict[str, torch.Tensor]] = field(init=False, factory=list)
    train_params: dict[str, torch.Tensor] = field(init=False)
    opt: torch.optim.AdamW = field(init=False)

    elapsed: stats.Elapsed = field(factory=stats.Elapsed, init=False)

    last_step: float = field(init=False)

    def run(self):
        asyncio.run(self.run_async())

    def serve_mode(self):
        self.train_params = {k: v.cpu() for (k, v) in self.model.state_dict().items()}
        self.model.to(device=self.config.device, dtype=self.config.serve_dtype)

    def train_mode(self):
        self.model.to(self.config.train_dtype).load_state_dict(self.train_params)

    def check_and_clear_save_request(self) -> bool:
        run_dir = self.config.run_dir
        if not run_dir:
            return False
        flagpath = os.path.join(run_dir, "SAVE_NOW")
        if os.path.exists(flagpath):
            os.unlink(flagpath)
            return True
        return False

    def train_step(self, batch):
        now = time.monotonic()
        rollout_time = now - self.last_step
        self.last_step = now

        self.replay_buffer.append(batch)
        if len(self.replay_buffer) > self.config.replay_buffer_steps:
            self.replay_buffer = self.replay_buffer[1:]

        self.train_mode()

        self.elapsed.step += 1

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
            "rollout_games": self.config.rollouts_per_step,
            "rollout_unique_plies": unique,
            "replay_buffer_plies": len(ds.flat_replay_buffer["positions"]),
            "train_step": self.elapsed.step,
            "rollout_time": rollout_time,
        }

        self.elapsed.epoch += 1
        train_start = time.monotonic()

        it = iter(ds)
        for i in range(0, self.config.train_positions, self.config.train_batch):
            try:
                self.elapsed.epoch += 1
                batch = next(it)
            except StopIteration:
                it = iter(ds)
                batch = next(it)

            self.opt.zero_grad()
            out = self.model(batch.inputs, *batch.extra_inputs)
            loss, metrics = loss_fn.loss_and_metrics(batch, out)
            loss.backward()
            self.opt.step()

            self.elapsed.positions += batch.inputs.size(0)

            if self.wandb is not None:
                self.wandb.log(
                    {
                        "train_loss": loss.item(),
                        "train_epoch": self.elapsed.epoch,
                        "positions": self.elapsed.positions,
                    }
                    | stats
                    | metrics
                )

        train_time = time.monotonic() - train_start
        print(
            f"step={self.elapsed.step}"
            f" games={self.config.rollouts_per_step}"
            f" plies={plies}"
            f" unique={unique}"
            f" rollout_time={rollout_time:0.2f}s"
            f" train_time={train_time:0.2f}s"
            f" ply/s={plies/(rollout_time):.1f}s"
            f" last_loss={loss.item():0.2f}"
        )

        if self.config.run_dir and (
            self.elapsed.step % self.config.save_freq == 0
            or self.elapsed.step == self.config.train_steps
            or self.check_and_clear_save_request()
        ):
            save_dir = os.path.join(
                self.config.run_dir, f"step_{self.elapsed.step:06d}"
            )
            print(f"Saving snapshot to {save_dir}...")
            self.save_snapshot(save_dir)
            latest_link = os.path.join(self.config.run_dir, "latest")
            try:
                os.unlink(latest_link)
            except FileNotFoundError:
                pass
            os.symlink(
                os.path.basename(save_dir),
                latest_link,
            )

        self.serve_mode()

    def save_snapshot(self, snapshot_path):
        os.makedirs(snapshot_path, exist_ok=True)
        loading.save_model(self.model, snapshot_path)
        torch.save(
            self.opt.state_dict(),
            os.path.join(snapshot_path, "opt.pt"),
        )
        torch.save(
            self.replay_buffer,
            os.path.join(snapshot_path, "replay_buffer.pt"),
        )
        with open(os.path.join(snapshot_path, "elapsed.yaml"), "w") as fh:
            yaml.dump(self.elapsed, fh)

    def should_exit(self):
        return self.elapsed.step >= self.config.train_steps

    async def command_loop(self):
        self.last_step = time.monotonic()
        while True:
            event = await self.loop.run_in_executor(None, self.shared.cmd.get)
            reply = None
            if isinstance(event, WaitForStartup):
                await self.ready.wait()
            elif isinstance(event, Shutdown):
                await self.server.stop(2)
                return
            elif isinstance(event, TrainStep):
                self.train_step(event.batch)
                reply = self.should_exit()
            else:
                raise AssertionError(f"Unknown command: {event}")
            await self.loop.run_in_executor(
                None, self.shared.reply.put, Reply(id=event.id, reply=reply)
            )

    def load_or_init_model(self):
        if self.config.run_dir:
            state_dir = os.path.join(self.config.run_dir, "latest")
            if os.path.exists(state_dir):
                loading.load_snapshot(self.model, state_dir)

                self.opt.load_state_dict(
                    torch.load(
                        os.path.join(state_dir, "opt.pt"),
                    )
                )
                self.replay_buffer = torch.load(
                    os.path.join(state_dir, "replay_buffer.pt"),
                )
                with open(os.path.join(state_dir, "elapsed.yaml"), "r") as fh:
                    self.elapsed = yaml.unsafe_load(fh)

                return

        if self.config.load_model:
            loading.load_snapshot(self.model, self.config.load_model)
        else:
            self.model.init_weights()

    async def run_async(self):
        if self.config.wandb:
            self.wandb = wandb.init(
                project=self.config.project, name=self.config.job_name
            )
            wandb.config.update(attrs.asdict(self.config))
        self.loop = asyncio.get_event_loop()

        self.opt = torch.optim.AdamW(self.model.parameters(), lr=self.config.lr)

        self.load_or_init_model()

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
            if self.wandb:
                self.wandb.finish()
