from tak import game, mcts


def test_mcts():
    engine = mcts.MCTS(
        config=mcts.Config(
            time_limit=0,
            simulation_limit=5,
            seed=1,
        )
    )

    print(engine.get_move(game.Position.from_config(game.Config(size=3))))
