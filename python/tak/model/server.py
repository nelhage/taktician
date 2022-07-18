import logging

from tak.proto import analysis_pb2_grpc
from tak.proto import analysis_pb2

import numpy as np
import torch
from torch import nn

from attrs import define, field

import asyncio
import time

import typing as T


MAX_QUEUE_DEPTH = 80


@define
class QueueRequest:
    position: torch.Tensor

    ready: asyncio.Event = field(factory=asyncio.Event)
    probs: T.Optional[np.ndarray] = None
    value: T.Optional[float] = None


@define
class Server(analysis_pb2_grpc.AnalysisServicer):
    model: nn.Module
    device: str = "cpu"

    queue: asyncio.Queue = field(factory=lambda: asyncio.Queue(MAX_QUEUE_DEPTH))

    async def worker_loop(self):
        loop = asyncio.get_running_loop()

        while True:
            batch = []
            batch.append(await self.queue.get())
            while True:
                if len(batch) >= 8:
                    try:
                        batch.append(self.queue.get_nowait())
                    except asyncio.QueueEmpty:
                        break
                else:
                    try:
                        elem = await asyncio.wait_for(self.queue.get(), 1.0 / 1000)
                        batch.append(elem)
                    except asyncio.TimeoutError:
                        break
            # now we have a batch

            @torch.inference_mode()
            def run_model(batch):
                positions = torch.zeros(
                    (len(batch), max(len(b.position) for b in batch)), dtype=torch.long
                )
                mask = torch.zeros_like(positions, dtype=torch.bool)
                for (i, b) in enumerate(batch):
                    positions[i, : len(b.position)] = b.position
                    mask[i, len(b.position) :].fill_(1)
                out = self.model(positions.to(self.device), mask.to(self.device))
                probs = (
                    torch.softmax(out["moves"], dim=-1)
                    .to(device="cpu", dtype=torch.float32)
                    .numpy()
                )
                values = out["values"].to(device="cpu", dtype=torch.float32).numpy()
                return (
                    probs,
                    values,
                )

            start = time.perf_counter()
            (probs, values) = await loop.run_in_executor(None, run_model, batch)
            end = time.perf_counter()
            logging.info(f"did batch len={len(batch)} dur={1000*(end-start):0.1f}ms")
            for (i, b) in enumerate(batch):
                b.probs = probs[i]
                b.value = values[i]
                b.ready.set()

    async def Evaluate(self, request, context):
        position = torch.tensor(request.position, dtype=torch.long)

        req = QueueRequest(position=position)
        await self.queue.put(req)
        await req.ready.wait()

        return analysis_pb2.EvaluateResponse(
            move_probs_bytes=req.probs.tobytes(), value=req.value
        )
