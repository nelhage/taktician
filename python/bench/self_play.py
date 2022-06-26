import argparse
import sys
import traceback

import tak
from tak import mcts
from tak.model import grpc
from attrs import define, field

import queue
from torch import multiprocessing

import torch
import numpy as np

import time

RESIGNATION_THRESHOLD = 0.95


@define
class Transcript:
    positions: list[tak.Position] = field(factory=list)
    moves: list[list[tak.Move]] = field(factory=list)
    probs: list[np.ndarray] = field(factory=list)


def self_play(engine, size=3):
    p = tak.Position.from_config(tak.Config(size=size))

    log = Transcript()

    tree = mcts.Node(position=p, move=None)

    while True:
        if abs(tree.v_zero) >= RESIGNATION_THRESHOLD:
            break

        color, over = tree.position.winner()
        if over is not None:
            break

        tree = engine.analyze_tree(tree)
        probs = engine.tree_probs(tree)

        log.positions.append(tree.position)
        log.moves.append([c.move for c in tree.children])
        log.probs.append(probs.numpy())

        tree = tree.children[torch.multinomial(probs, 1).item()]

    return log


@define
class Job:
    size: int

    host: str
    port: int

    simulations: int

    sema: multiprocessing.Semaphore
    queue: multiprocessing.Queue
    shutdown: multiprocessing.Event


def entrypoint(job: Job, id: int):
    try:
        run_job(job, id)
        job.queue.close()
        job.queue.join_thread()
        job.shutdown.wait()
    except Exception as ex:
        print(f"[{id}] Process crashed: {ex}", file=sys.stderr)
        traceback.print_exc(file=sys.stderr)


def run_job(job: Job, id: int):
    network = grpc.GRPCNetwork(host=job.host, port=job.port)

    engine = mcts.MCTS(
        mcts.Config(
            network=network,
            simulation_limit=job.simulations,
            time_limit=0,
        )
    )

    logs = []

    while True:
        if not job.sema.acquire(block=False):
            break
        logs.append(self_play(engine, job.size))

    job.queue.put(logs)


def main(argv):
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--simulations",
        dest="simulations",
        type=int,
        default=100,
        metavar="POSITIONS",
    )
    parser.add_argument(
        "--size",
        dest="size",
        type=int,
        default=3,
        metavar="SIZE",
    )
    parser.add_argument(
        "--host",
        type=str,
        default="localhost",
    )
    parser.add_argument(
        "--port",
        type=int,
        default=5001,
    )
    parser.add_argument(
        "--games",
        type=int,
        default=1,
    )
    parser.add_argument(
        "--threads",
        type=int,
        default=1,
    )

    args = parser.parse_args(argv)

    job = Job(
        size=args.size,
        simulations=args.simulations,
        host=args.host,
        port=args.port,
        sema=multiprocessing.Semaphore(value=args.games),
        queue=multiprocessing.Queue(maxsize=args.threads),
        shutdown=multiprocessing.Event(),
    )

    start = time.time()
    processes = [
        multiprocessing.Process(target=entrypoint, args=(job, i))
        for i in range(args.threads)
    ]
    for p in processes:
        p.start()

    results = []
    while len(results) < args.threads:
        try:
            result = job.queue.get(block=True, timeout=1)
            results.append(result)
        except queue.Empty:
            for p in processes:
                if p.exitcode not in [0, None]:
                    raise RuntimeError("Process crashed!")

    job.shutdown.set()

    for p in processes:
        p.join()
    end = time.time()

    logs = [l for r in results for l in r]

    print(
        f"done games={len(logs)} plies={sum(len(l.positions) for l in logs)} threads={args.threads} duration={end-start:.2f} games/s={args.games/(end-start):.1f}"
    )


if __name__ == "__main__":
    main(sys.argv[1:])
