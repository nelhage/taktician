import torch

import xformer


def test_xformer():
    cfg = xformer.Config(
        n_vocab=256,
        d_model=2 * 128,
        n_layer=3,
        d_head=32,
    )
    model = xformer.Transformer(cfg)

    tokens = torch.randint(0, cfg.n_vocab, (4, 1024))
    logits = model(tokens)

    assert logits.shape == (4, 1024, cfg.n_vocab)
    assert logits.isnan().sum() == 0, "No NaNs"
