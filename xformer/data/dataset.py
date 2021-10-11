import torch

class PTDataset(torch.utils.data.IterableDataset):
  def __init__(self, files):
    self.files = files

  def __iter__(self):
    for file in self.files:
      data = torch.load(file)
      yield from data
