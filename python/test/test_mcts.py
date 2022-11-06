from tak import game, mcts
import xformer
from tak.model import heads, wrapper

import torch
import tak_ext


def test_mcts():
    cfg = xformer.Config(
        n_layer=1,
        d_model=64,
        d_head=32,
        n_ctx=128,
        n_vocab=256,
        autoregressive_mask=False,
        output_head=heads.PolicyValue,
    )
    model = xformer.Transformer(cfg)

    engine = mcts.MCTS(
        config=mcts.Config(
            time_limit=0,
            simulation_limit=5,
        ),
        network=wrapper.ModelWrapper(model),
    )

    print(engine.get_move(game.Position.from_config(game.Config(size=3))))


def test_solve_policy():
    pi_theta = torch.tensor(
        [0.1818, 0.1651, 0.1377, 0.1367, 0.1307, 0.1033, 0.0655, 0.0558, 0.0235]
    )
    q = torch.tensor(
        [-0.6232, 0.6529, 0.6529, 0.6529, 0.6529, 0.6529, 0.6529, 0.6529, 0.6529]
    )
    lambda_n = 0.0899954085146515

    policy = tak_ext.solve_policy(pi_theta, q, lambda_n)

    py_policy = mcts.solve_policy_python(pi_theta, q, lambda_n)

    assert (policy >= 0).all()
    assert ((policy - py_policy).abs() <= 1e-2).all()
