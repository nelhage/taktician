import xformer.data
from xformer.data import record_file
import argparse
import zstandard as zstd
import pickle
import torch
import os
import io
import glob
import re
import random
import multiprocessing
import contextlib

class ChunkWriter:
  def __init__(self, output_files):
    self.handles = [
      record_file.Writer(open(fh, 'wb')) for fh in output_files
    ]

  def write(self, obj):
    handle = random.choice(self.handles)
    handle.write(pickle.dumps(obj))

  def close(self):
    for fh in self.handles:
      fh.close()


def postprocess_chunk(infile, outfile):
  with record_file.Reader(open(infile, 'rb')) as read:
    records = [pickle.loads(data) for data in read]
  random.shuffle(records)
  for r in records:
    r['text'] = torch.tensor(bytearray(r['text']), dtype=torch.uint8)
  torch.save(records, outfile)
  os.unlink(infile)

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

  def want(dataset):
    if set_filter is None:
      return True
    cached = want_cache.get(dataset, None)
    if cached is not None:
      return cached
    match = bool(set_filter.search(dataset))
    set_filter[dataset] = match
    return match


  bytes_read = 0
  read_buffer = b''

  temp_files = [
    args.output + f"{i:04d}.tmp" for i in range(args.n_chunks)
  ]
  out_files = [
    args.output + f"{i:04d}.pt" for i in range(args.n_chunks)
  ]

  writer = ChunkWriter(temp_files)

  with contextlib.closing(writer):
    for file in files:
      for record in xformer.data.pile_iterator(file):
        if not want(record['meta']['pile_set_name']):
          continue
        data = read_buffer + record['text'].encode('utf-8')
        for i in range(0, len(data) - args.n_ctx + 1, args.n_ctx):
          assert len(data) >= i + args.n_ctx
          writer.write({'text': data[i:i+args.n_ctx]})
          bytes_read += args.n_ctx
          if bytes_read > args.read_bytes:
            break
        read_buffer = data[i:]
        if bytes_read > args.read_bytes:
          break

  pool = multiprocessing.Pool()
  pool.starmap(postprocess_chunk, zip(temp_files, out_files))

if __name__ == '__main__':
  main()
