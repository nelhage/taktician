import xformer
import xformer.data
import torch
import time
import itertools
import argparse
import wandb


def main():
  parser = argparse.ArgumentParser(description="Train a transformer")
  parser.add_argument('--layers', type=int, default=2, help="Number of layers")
  parser.add_argument('--d_model', type=int, default=None, help="embedding dimension")
  parser.add_argument('--d_head', type=int, default=32, help="head dimension")
  parser.add_argument('--n_ctx', type=int, default=1024, help="maximum context length")
  parser.add_argument('--data', type=str, default='data/pile/train/00.jsonl.zst', help="datasource")
  parser.add_argument('--batch', type=int, default=64, help="batch size")
  parser.add_argument('--minibatch', type=int, default=4, help="minibatch")
  parser.add_argument('--device', type=str, choices=('cpu', 'cuda'), default='cuda', help="device")
  parser.add_argument('--wandb', action='store_true', default=False)
  parser.add_argument('--no-wandb', action='store_false', dest='wandb')

  args = parser.parse_args()

  cfg = xformer.Config(
    n_layer = args.layers,
    d_model = args.d_model or 128*args.layers,
    d_head = args.d_head,
    n_ctx = args.n_ctx,
    n_vocab = 256,
  )

  ds = xformer.data.PileDataset(args.data, n_ctx=cfg.n_ctx)
  loader = torch.utils.data.DataLoader(ds, batch_size=args.minibatch, collate_fn=xformer.data.collate_fn)
  model = xformer.Transformer(cfg, dtype=torch.float32, device=args.device)

  xent = torch.nn.CrossEntropyLoss(reduction='mean')
  opt = torch.optim.Adam(model.parameters())

  steps_per_batch = args.batch // args.minibatch

  data = iter(loader)

  if args.wandb:
    run = wandb.init()
    wandb.watch(model, log_freq=10, log='all')
    wandb.config.update(args)

  start = time.time()
  tokens = 0

  for step_i in itertools.count():
    avg_loss = 0.0
    opt.zero_grad()
    for _ in range(steps_per_batch):
      batch = next(data)
      batch = batch.to(args.device)
      logits = model(batch[:, :-1])
      targets = batch[:, 1:]
      loss = xent(logits.permute(0, 2, 1), targets)
      avg_loss += loss.item()
      tokens += batch.numel()
      loss.backward()
    opt.step()
    now = time.time()
    print(f"[step={step_i:06d} t={now-start:.1f}s tokens={tokens:08d}] loss={avg_loss/steps_per_batch:2.2f}")
    if args.wandb:
      wandb.log({
        'tokens': tokens,
        'elapsed_time': now-start,
        'train_loss': avg_loss,
      }, step=step_i)

if __name__ == '__main__':
  main()
