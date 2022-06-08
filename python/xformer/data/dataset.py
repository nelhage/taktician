import torch

class PTDataset(torch.utils.data.IterableDataset):
  def __init__(self, files):
    self.files = files

  def __iter__(self):
    for file in self.files:
      data = torch.load(file)
      for record in data:
        if 'text' in record:
          record['text'] = record['text'].to(torch.long)
        yield record
