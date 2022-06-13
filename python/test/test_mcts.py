from tak import game, mcts
import xformer
from tak.model import heads, wrapper


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
            seed=1,
            network=wrapper.ModelWrapper(model),
        )
    )

    print(engine.get_move(game.Position.from_config(game.Config(size=3))))
