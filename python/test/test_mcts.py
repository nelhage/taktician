from tak import mcts, game


def test_mcts():
    engine = mcts.MCTS(
        config=mcts.Config(
            time_limit=0,
            position_limit=5,
            seed=1,
        )
    )

    print(engine.get_move(game.Position.from_config(game.Config(size=3))))
