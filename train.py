import xformer
import xformer.data
import torch
import time
import itertools

BATCH_SIZE = 512
MINIBATCH_SIZE = 4

def main():
  cfg = xformer.Config(
    n_layer = 4,
    d_model = 4 * 128,
    d_head = 32,
    n_vocab = 256,
  )
  ds = xformer.data.PileDataset('data/pile/train/00.jsonl.zst', n_ctx=cfg.n_ctx)
  loader = torch.utils.data.DataLoader(ds, batch_size=MINIBATCH_SIZE, collate_fn=xformer.data.collate_fn)
  model = xformer.Transformer(cfg, dtype=torch.float32, device='cuda')

  xent = torch.nn.CrossEntropyLoss(reduction='mean')
  opt = torch.optim.Adam(model.parameters())

  steps_per_batch = BATCH_SIZE // MINIBATCH_SIZE

  data = iter(loader)

  start = time.time()

  for step_i in itertools.count():
    avg_loss = 0.0
    for _ in range(steps_per_batch):
      batch = next(data)
      batch = batch.cuda()
      logits = model(batch[:, :-1])
      targets = batch[:, 1:]
      loss = xent(logits.permute(0, 2, 1), targets)
      avg_loss += loss.item()
      opt.zero_grad()
      loss.backward()
      opt.step()
    now = time.time()
    print(f"[step={step_i:06d} t={now-start:6.1f}s tokens={(step_i+1)*BATCH_SIZE*cfg.n_ctx:08d}] loss={avg_loss/steps_per_batch:2.2f}")

if __name__ == '__main__':
  main()
