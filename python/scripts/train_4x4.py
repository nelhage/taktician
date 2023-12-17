from tak.alphazero import cli, hooks, trainer, schedule
import os.path
import yaml
import shlex


def main():
    parser = cli.build_parser()
    parser.add_argument(
        "--warmup-steps",
        type=int,
        default=100,
    )
    parser.set_defaults(
        rollouts_per_step=200,
        rollout_workers=50,
        train_positions=16 * 1024,
        replay_buffer_steps=16,
        batch=256,
        lr=1e-4,
        test_freq=50,
        eval_freq=50,
        size=4,
        rollout_ply_limit=40,
        noise_alpha=0.5,
    )

    args = parser.parse_args()
    if args.job_name and not args.run_dir:
        args.run_dir = os.path.join(cli.ROOT, "data/alphazero." + args.job_name)
        print(f"Saving run to {args.run_dir}...")

    if args.run_dir is not None and os.path.exists(
        os.path.join(args.run_dir, "run.yaml")
    ):
        with open(os.path.join(args.run_dir, "run.yaml")) as fh:
            run = yaml.unsafe_load(fh)
    else:
        run = cli.build_train_run(args)

        if args.warmup_steps > 0:
            run.lr_schedule = schedule.LinearWarmup(
                warmup_steps=args.warmup_steps, final_value=run.lr
            )

        run.hooks.append(
            hooks.EvalHook(
                name="tako3",
                opponent="taktician tei -depth=3 -table-mem=-1",
                frequency=args.eval_freq,
                openings=os.path.join(cli.ROOT, "data/4x4-openings"),
            )
        )
        run.hooks.append(
            hooks.EvalHook(
                name="tako5",
                opponent="taktician tei -depth=5 -table-mem=-1",
                frequency=2 * args.eval_freq,
                openings=os.path.join(cli.ROOT, "data/4x4-openings"),
            )
        )
    # run.hooks.append(
    #     hooks.EvalHook(
    #         name="step8k",
    #         opponent=shlex.join(
    #             [
    #                 os.path.join(cli.ROOT, "python/scripts/tei"),
    #                 "--host",
    #                 "localhost",
    #                 "--port",
    #                 "50005",
    #                 "--simulation-limit=1",
    #                 "--argmax",
    #             ]
    #         ),
    #         model=(os.path.join(cli.ROOT, "data/size-4/step_008000/"), 50005),
    #         frequency=args.eval_freq,
    #         openings=os.path.join(cli.ROOT, "data/4x4-openings"),
    #     )
    # armup)
    trainer.TrainingRun(config=run).run()


if __name__ == "__main__":
    main()
