import typing as T  # noqa
import time
import os
import itertools

import torch
import queue

import grpc
import asyncio
import threading

import io
import zstandard


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


def load_state(state: TrainState, snapshot_path: str):
    loading.load_snapshot(state.model, snapshot_path)

    state.opt.load_state_dict(
        torch.load(
            os.path.join(snapshot_path, "opt.pt"),
        )
    )
    rbpath = os.path.join(snapshot_path, "replay_buffer.pt")
    if os.path.exists(rbpath):
        state.replay_buffer = torch.load(rbpath)
    else:
        with open(rbpath + ".zst", "rb") as fh:
            cctx = zstandard.ZstdDecompressor()
            zreader = cctx.stream_reader(fh)
            data = zreader.read()
            state.replay_buffer = torch.load(io.BytesIO(data))

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


class Scheduler:
    def value(self, state: TrainState) -> float:
        ...


def dedup_batch(batch):
    N = batch["positions"].shape[0]
    out = {k: torch.zeros_like(v) for (k, v) in batch.items()}
    ids = {}
    counts = torch.zeros(N)
    next = 0

    keys = [k for k in batch if k not in ["positions", "mask"]]

    for i in range(N):
        key = tuple(batch["positions"][i][batch["mask"][i]].tolist())
        if key in ids:
            idx = ids[key]
        else:
            idx = next
            next += 1
            ids[key] = idx
            out["positions"][idx] = batch["positions"][i]
            out["mask"][idx] = batch["mask"][i]
        counts[idx] += 1
        for k in keys:
            out[k][idx] += batch[k][i]

    for k in keys:
        out[k] /= counts.reshape((-1,) + (1,) * (len(out[k].shape) - 1))
    return {k: v[:next] for (k, v) in out.items()}


@define
class TrainingRun:
    config: Config

    state: TrainState = field(init=False)
    train_params: dict[str, torch.Tensor] = field(init=False)

    serve_thread: threading.Thread = field(init=False)

    def serve_mode(self):
        self.train_params = {
            k: v.cpu() for (k, v) in self.state.model.state_dict().items()
        }
        self.state.model.to(device=self.config.device, dtype=self.config.serve_dtype)

    def train_mode(self):
        self.state.model.to(self.config.train_dtype).load_state_dict(self.train_params)

    def train_step(self, batch):
        pre_dedup = len(batch["positions"])

        batch = dedup_batch(batch)

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

        plies = len(batch["positions"])
        self.state.step_stats.update(
            {
                "rollout_plies": plies,
                "rollout_games": self.config.rollouts_per_step,
                "rollout_total_plies": pre_dedup,
                "replay_buffer_plies": len(ds.flat_replay_buffer["positions"]),
            }
        )

        self.state.elapsed.epoch += 1

        if self.config.lr_schedule:
            lr = self.config.lr_schedule.value(self.state)
            self.state.step_stats["lr"] = lr
            for grp in self.state.opt.param_groups:
                grp["lr"] = lr

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
            opt_path = os.path.join(self.config.load_model, "opt.pt")
            if os.path.exists(opt_path):
                self.state.opt.load_state_dict(torch.load(opt_path))
        else:
            self.state.model.init_weights()

    def train_loop(self):
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

        def fmt(v):
            if isinstance(v, float):
                return f"{v:.3f}"
            return str(v)

        try:
            for hook in self.config.hooks:
                hook.before_run(self.state, self.config)

            while not self.should_exit():
                self.state.step_stats = {}

                for hook in self.config.hooks:
                    hook.before_rollout(self.state)

                logs = rollout_engine.play_many(self.config.rollouts_per_step)
                batch = self_play.encode_games(logs)
                batch["positions"] = batch["positions"].to(torch.long)

                for hook in self.config.hooks:
                    hook.before_train(self.state)

                self.train_step(batch)

                for hook in self.config.hooks:
                    hook.after_step(self.state)
                for hook in self.config.hooks:
                    hook.finalize(self.state)

                print(
                    f"step={self.state.elapsed.step} "
                    + " ".join(
                        f"{key}={fmt(value)}"
                        for (key, value) in self.state.step_stats.items()
                    )
                )

            for hook in self.config.hooks:
                hook.after_run(self.state)

        finally:
            rollout_engine.stop()

    def run(self):
        model = xformer.Transformer(self.config.model, device=self.config.device)
        self.state = TrainState(
            model=model,
            opt=torch.optim.AdamW(model.parameters(), lr=self.config.lr),
        )

        self.load_or_init_model()
        if self.config.run_dir:
            config_path = os.path.join(self.config.run_dir, "run.yaml")
            os.makedirs(os.path.dirname(config_path), exist_ok=True)
            with open(config_path, "w") as fh:
                yaml.dump(self.config, fh)

        self.serve_mode()

        os.environ["GRPC_ENABLE_FORK_SUPPORT"] = "0"

        ready = threading.Event()

        async def serve():
            loop = asyncio.get_running_loop()
            server = grpc.aio.server()
            server.add_insecure_port(f"localhost:{self.config.server_port}")
            analysis = tak.model.server.Server(
                model=self.state.model, device=self.config.device
            )
            tasks = [asyncio.create_task(analysis.worker_loop())]

            analysis_pb2_grpc.add_AnalysisServicer_to_server(
                analysis,
                server,
            )

            await server.start()
            await loop.run_in_executor(None, ready.set)
            done, pending = await asyncio.wait(
                tasks + [asyncio.create_task(server.wait_for_termination())],
                return_when=asyncio.FIRST_COMPLETED,
            )
            for task in pending:
                task.cancel()
            for task in done:
                task.result()

        self.serve_thread = threading.Thread(
            target=asyncio.run, args=(serve(),), daemon=True
        )
        self.serve_thread.start()

        self.train_loop()
