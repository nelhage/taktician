import xformer.data
from xformer.data import record_file
import argparse
import zstandard as zstd
import pickle
import torch
import os
import itertools
import io
import glob
import re
import random
import multiprocessing
import contextlib

class BufferedTorchWriter:
  def __init__(self, writer: record_file.Writer, flush_interval=100):
    self.writer = writer
    self.flush_interval = flush_interval
    self.buffer = []

  def write(self, obj):
    self.buffer.append(obj)
    if len(self.buffer) >= self.flush_interval:
      self.flush()

  def flush(self):
    bytes = io.BytesIO()
    torch.save(self.buffer, bytes)
    self.writer.write(bytes.getvalue())
    self.buffer.clear()

  def close(self):
    self.flush()
    self.writer.close()

class ChunkWriter:
  def __init__(self, output_files):
    self.handles = [
      BufferedTorchWriter(record_file.Writer(open(fh, 'wb')))
      for fh in output_files
    ]

  def write(self, obj):
    handle = random.choice(self.handles)
    handle.write(obj)

  def close(self):
    for fh in self.handles:
      fh.close()


def postprocess_chunk(infile, outfile):
  with record_file.Reader(open(infile, 'rb')) as read:
    records = []
    for data in read:
      records += torch.load(io.BytesIO(data))
  random.shuffle(records)
  torch.save(records, outfile)
  os.unlink(infile)


def filtered_texts(input_texts, want, limit_bytes):
  read_bytes = 0
  for record in input_texts:
    if not want(record):
      continue
    text = record['text'].encode('utf-8')
    yield text
    read_bytes += len(text)
    if read_bytes >= limit_bytes:
      return

def join_batches(texts, n_ctx):
  current = None
  text_nos = None
  idx = 0
  text_no = 0

  for txt in texts:
    while len(txt) > 0:
      if current is None:
        current = torch.zeros(n_ctx, dtype=torch.uint8)
        text_nos = torch.zeros(n_ctx, dtype=torch.uint8)
        text_no = 1
        idx = 0

      take = min(len(current) - idx - 1, len(txt))
      text_nos[idx:idx+take+1] = text_no
      text_no += 1
      current[idx] = 0
      current.numpy()[idx+1:idx+1+take] = memoryview(txt[:take])
      txt = txt[take:]
      idx += take+1

      if idx >= len(current)-1:
        yield {'text': current, 'text_nos': text_nos}
        current = None
  if current is not None:
    yield {'text': current, 'text_nos': text_nos}

def main():
  parser = argparse.ArgumentParser(description="Split a pile dataset into chunks")
  parser.add_argument('--input', type=str, default='data/pile/train/*.zst', help="Input files")
  parser.add_argument('--output', type=str, default='data/pile/chunked/train-', help="Prefix for output files")
  parser.add_argument('--n_chunks', type=int, default=64, help="Number of output chunks")
  parser.add_argument('--n_ctx', type=int, default=1024, help="Context length in bytes")
  parser.add_argument('--set_filter', type=str, default=None, help="regex to filter pile set")
  parser.add_argument('--read_bytes', type=int, default=1024*1024*1024, help="Number of bytes to read")

  args = parser.parse_args()

  files = glob.glob(args.input)
  if len(files) == 0:
    raise ValueError(f"No files found: {args.input}")

  set_filter = None
  if args.set_filter:
    set_filter = re.compile(args.set_filter)
  want_cache = {}

  def want(record):
    if set_filter is None:
      return True
    dataset = record['meta']['pile_set_name']
    cached = want_cache.get(dataset, None)
    if cached is not None:
      return cached
    match = bool(set_filter.search(dataset))
    set_filter[dataset] = match
    return match

  temp_files = [
    args.output + f"{i:04d}.tmp" for i in range(args.n_chunks)
  ]
  out_files = [
    args.output + f"{i:04d}.pt" for i in range(args.n_chunks)
  ]

  writer = ChunkWriter(temp_files)

  in_texts = itertools.chain.from_iterable(
    xformer.data.pile_iterator(file)
    for file in files)

  want_texts = filtered_texts(in_texts, want, args.read_bytes)
  batches = join_batches(want_texts, args.n_ctx)

  with contextlib.closing(writer):
    for batch in batches:
        writer.write(batch)

  pool = multiprocessing.Pool()
  pool.starmap(postprocess_chunk, zip(temp_files, out_files))

if __name__ == '__main__':
  main()
