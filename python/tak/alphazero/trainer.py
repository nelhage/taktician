import typing as T  # noqa
import time
import os
import itertools

import torch
import queue
from torch import multiprocessing

import grpc
import asyncio


import xformer
from xformer import loading
import yaml

from tak.proto import analysis_pb2_grpc
import tak.model.server
from tak.model import batches, losses
from tak import self_play

from attrs import field, define
import attrs

from .. import Config
from . import data, stats


@define(slots=False)
class TrainState:
    model: xformer.Transformer
    opt: torch.optim.AdamW
    elapsed: stats.Elapsed = field(factory=stats.Elapsed, init=False)
    replay_buffer: list[dict[str, torch.Tensor]] = field(init=False, factory=list)

    step_stats: dict = field(init=False, factory=dict)


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


def load_state(state: TrainState, snapshot_path: str):
    loading.load_snapshot(state.model, snapshot_path)

    state.opt.load_state_dict(
        torch.load(
            os.path.join(snapshot_path, "opt.pt"),
        )
    )
    state.replay_buffer = torch.load(
        os.path.join(snapshot_path, "replay_buffer.pt"),
    )
    with open(os.path.join(snapshot_path, "elapsed.yaml"), "r") as fh:
        state.elapsed = yaml.unsafe_load(fh)


class Hook:
    def before_run(self, state: TrainState, config: Config):
        pass

    def after_run(self, state: TrainState):
        pass

    def before_rollout(self, state: TrainState):
        pass

    def before_train(self, state: TrainState):
        pass

    def after_step(self, state: TrainState):
        pass

    def finalize(self, state: TrainState):
        pass


@define
class TrainingRun:
    config: Config

    state: TrainState = field(init=False)
    train_params: dict[str, torch.Tensor] = field(init=False)

    loop: asyncio.BaseEventLoop = field(init=False)
    server: grpc.aio.Server = field(init=False)
    tasks: list[asyncio.Task] = field(init=False, factory=list)

    def run(self):
        asyncio.run(self.run_async())

    def serve_mode(self):
        self.train_params = {
            k: v.cpu() for (k, v) in self.state.model.state_dict().items()
        }
        self.state.model.to(device=self.config.device, dtype=self.config.serve_dtype)

    def train_mode(self):
        self.state.model.to(self.config.train_dtype).load_state_dict(self.train_params)

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
        self.state.replay_buffer.append(batch)
        if len(self.state.replay_buffer) > self.config.replay_buffer_steps:
            self.state.replay_buffer = self.state.replay_buffer[1:]

        self.train_mode()

        self.state.elapsed.step += 1

        loss_fn = losses.PolicyValue()
        ds = data.ReplayBufferDataset(
            replay_buffer=self.state.replay_buffer,
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
        self.state.step_stats.update(
            {
                "rollout_plies": plies,
                "rollout_games": self.config.rollouts_per_step,
                "rollout_unique_plies": unique,
                "replay_buffer_plies": len(ds.flat_replay_buffer["positions"]),
            }
        )

        self.state.elapsed.epoch += 1

        it = iter(ds)
        for i in range(0, self.config.train_positions, self.config.train_batch):
            try:
                self.state.elapsed.epoch += 1
                batch = next(it)
            except StopIteration:
                it = iter(ds)
                batch = next(it)

            self.state.opt.zero_grad()
            out = self.state.model(batch.inputs, *batch.extra_inputs)
            loss, metrics = loss_fn.loss_and_metrics(batch, out)
            loss.backward()
            self.state.opt.step()

            self.state.elapsed.positions += batch.inputs.size(0)

            if i == 0:
                self.state.step_stats["train_loss.before"] = loss.item()
        self.state.step_stats["train_loss"] = loss.item()
        self.state.step_stats.update(metrics)

        print(
            f"step={self.state.elapsed.step}"
            f" games={self.config.rollouts_per_step}"
            f" plies={plies}"
            f" unique={unique}"
            #            f" rollout_time={rollout_time:0.2f}s"
            #            f" train_time={train_time:0.2f}s"
            #            f" step_time={step_time:0.2f}s"
            #            f" ply/s={plies/(rollout_time):.1f}s"
            f" last_loss={loss.item():0.2f}"
        )

        if self.config.run_dir and (
            self.state.elapsed.step % self.config.save_freq == 0
            or self.state.elapsed.step == self.config.train_steps
            or self.check_and_clear_save_request()
        ):
            save_dir = os.path.join(
                self.config.run_dir, f"step_{self.state.elapsed.step:06d}"
            )
            print(f"Saving snapshot to {save_dir}...")
            save_snapshot(self.state, save_dir)
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

    def should_exit(self):
        return self.state.elapsed.step >= self.config.train_steps

    def load_or_init_model(self):
        if self.config.run_dir:
            state_dir = os.path.join(self.config.run_dir, "latest")
            if os.path.exists(state_dir):
                load_state(self.state, state_dir)
                return

        if self.config.load_model:
            loading.load_snapshot(self.state.model, self.config.load_model)
        else:
            self.state.model.init_weights()

    async def train_loop(self):
        rollout_engine = self_play.MultiprocessSelfPlayEngine(
            config=self_play.SelfPlayConfig(
                size=self.config.size,
                workers=self.config.rollout_workers,
                resignation_threshold=self.config.rollout_resignation_threshold,
                ply_limit=self.config.rollout_ply_limit,
                engine_factory=self_play.BuildRemoteMCTS(
                    host="localhost",
                    port=self.config.server_port,
                    config=self.config.rollout_config,
                ),
            )
        )

        try:
            for hook in self.config.hooks:
                hook.before_run(self.state, self.config)

            while not self.should_exit():
                self.state.step_stats = {}

                for hook in self.config.hooks:
                    hook.before_rollout(self.state)

                logs = await self.loop.run_in_executor(
                    None, rollout_engine.play_many, self.config.rollouts_per_step
                )
                batch = self_play.encode_games(logs)
                batch["positions"] = batch["positions"].to(torch.long)

                for hook in self.config.hooks:
                    hook.before_train(self.state)

                self.train_step(batch)
                for hook in self.config.hooks:
                    hook.after_step(self.state)
                for hook in self.config.hooks:
                    hook.finalize(self.state)

            for hook in self.config.hooks:
                hook.after_run(self.state)

        finally:
            rollout_engine.stop()

    async def run_async(self):
        model = xformer.Transformer(self.config.model, device=self.config.device)
        self.state = TrainState(
            model=model,
            opt=torch.optim.AdamW(model.parameters(), lr=self.config.lr),
        )

        self.loop = asyncio.get_event_loop()

        self.load_or_init_model()
        if self.config.run_dir:
            config_path = os.path.join(self.config.run_dir, "run.yaml")
            os.makedirs(os.path.dirname(config_path), exist_ok=True)
            with open(config_path, "w") as fh:
                yaml.dump(self.config, fh)

        self.serve_mode()

        multiprocessing.set_start_method("spawn")
        os.environ["GRPC_ENABLE_FORK_SUPPORT"] = "0"

        self.server = grpc.aio.server()
        self.server.add_insecure_port(f"localhost:{self.config.server_port}")

        analysis = tak.model.server.Server(
            model=self.state.model, device=self.config.device
        )

        self.tasks.append(asyncio.create_task(analysis.worker_loop()))

        analysis_pb2_grpc.add_AnalysisServicer_to_server(
            analysis,
            self.server,
        )
        await self.server.start()
        train_task = asyncio.create_task(self.train_loop())
        self.tasks.append(train_task)
        done, pending = await asyncio.wait(
            self.tasks + [self.server.wait_for_termination()],
            return_when=asyncio.FIRST_COMPLETED,
        )
        await self.server.stop(0)
        for task in pending:
            task.cancel()
        for task in done:
            task.result()
