from .model_pb2 import ModelDef
from .corpus_entry_pb2 import CorpusEntry
from .taktician_pb2 import AnalyzeRequest, AnalyzeResponse, \
  IsPositionInTakRequest, IsPositionInTakResponse
from .taktician_pb2_grpc import TakticianStub, TakticianServicer
