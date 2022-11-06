from . import encoding
from attrs import define, field

import numpy as np
import torch

from tak.proto import analysis_pb2, analysis_pb2_grpc
import grpc


@define
class GRPCNetwork:
    host: str
    port: int
    stub: analysis_pb2_grpc.AnalysisStub = field(init=False)

    def __attrs_post_init__(self):
        channel = grpc.insecure_channel(f"{self.host}:{self.port}")
        self.stub = analysis_pb2_grpc.AnalysisStub(channel)

    def evaluate(self, pos):
        with torch.no_grad():
            encoded = encoding.encode(pos)
            out = self.stub.Evaluate(analysis_pb2.EvaluateRequest(position=encoded))
            move_probs = torch.from_numpy(
                np.frombuffer(out.move_probs_bytes, dtype=np.float32).copy()
            )
            return move_probs, out.value
