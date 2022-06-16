from . import encoding
from attrs import define, field

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
            return torch.tensor(out.move_probs), out.value
