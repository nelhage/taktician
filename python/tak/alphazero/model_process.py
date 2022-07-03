import typing as T  # noqa

import torch
from torch import multiprocessing

import grpc
from tak.proto import analysis_pb2_grpc
import asyncio

import xformer

import tak.model.server

from attrs import field, define

from .. import Config


class Command:
    pass


class Shutdown(Command):
    pass


@define
class ModelServerShared:
    ready: multiprocessing.Event = field(factory=multiprocessing.Event, init=False)
    cmd: multiprocessing.Queue = field(factory=multiprocessing.Queue, init=False)


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
        worker = ModelServerProcess(
            model=self.model,
            config=self.config,
            shared=self.shared,
        )
        worker.run()

    def start(self):
        self.process.start()
        self.shared.ready.wait()

    def stop(self):
        self.shared.cmd.put(Shutdown())
        self.process.join()


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

    loop: asyncio.BaseEventLoop = field(init=False)
    server: grpc.aio.Server = field(init=False)
    tasks: list[asyncio.Task] = field(init=False, factory=list)

    def run(self):
        asyncio.run(self.run_async())

    async def command_loop(self):
        while True:
            event = await self.loop.run_in_executor(None, self.shared.cmd.get)
            if isinstance(event, Shutdown):
                await self.server.stop(2)
                return

    async def run_async(self):
        self.loop = asyncio.get_event_loop()

        self.model.to(device=self.config.device, dtype=torch.float16)

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
        await self.loop.run_in_executor(None, self.shared.ready.set)
        await self.server.wait_for_termination()
        for task in self.tasks:
            task.cancel()
