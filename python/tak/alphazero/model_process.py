import typing as T  # noqa

import torch
from torch import multiprocessing

import grpc
from tak.proto import analysis_pb2_grpc
import asyncio

import xformer

import tak.model.server

from attrs import field, define


@define
class ModelServerShared:
    shutdown: multiprocessing.Event = field(factory=multiprocessing.Event, init=False)
    ready: multiprocessing.Event = field(factory=multiprocessing.Event, init=False)


@define
class ModelServerHandle:
    model: xformer.Transformer
    device: str
    port: int
    process: multiprocessing.Process = field(init=False)
    shared: ModelServerShared = field(factory=ModelServerShared, init=False)

    def __attrs_post_init__(self):
        self.process = multiprocessing.Process(
            target=self._run_in_spawn, name="analysis_server"
        )

    def _run_in_spawn(self):
        worker = ModelServerProcess(
            model=self.model,
            device=self.device,
            port=self.port,
            shared=self.shared,
        )
        worker.run()

    def start(self):
        self.process.start()
        self.shared.ready.wait()

    def stop(self):
        self.shared.shutdown.set()
        self.process.join()


def create_server(
    model: xformer.Transformer, device: str = "cpu", port: int = 5001
) -> ModelServerHandle:
    return ModelServerHandle(
        model=model,
        device=device,
        port=port,
    )


@define
class ModelServerProcess:
    model: xformer.Transformer
    device: str
    port: int
    shared: ModelServerShared

    loop: asyncio.BaseEventLoop = field(init=False)
    server: grpc.aio.Server = field(init=False)

    def run(self):
        asyncio.run(self.run_async())

    async def handle_shutdown(self):
        await self.loop.run_in_executor(None, self.shared.shutdown.wait)
        await self.server.stop(5)

    async def run_async(self):
        self.loop = asyncio.get_event_loop()

        model = self.model.to(device=self.device, dtype=torch.float16)

        self.server = grpc.aio.server()
        self.server.add_insecure_port(f"localhost:{self.port}")

        analysis = tak.model.server.Server(model=model, device=self.device)
        worker = asyncio.create_task(analysis.worker_loop())

        asyncio.create_task(self.handle_shutdown())

        analysis_pb2_grpc.add_AnalysisServicer_to_server(
            analysis,
            self.server,
        )
        await self.server.start()
        await self.loop.run_in_executor(None, self.shared.ready.set)
        await self.server.wait_for_termination()
        worker.cancel()
