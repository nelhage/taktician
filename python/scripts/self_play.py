import argparse
import sys
import traceback
from typing import Optional

import tak
from tak import mcts
from tak.model import grpc, encoding
from attrs import define, field
import tqdm

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
    values: list[float] = field(factory=list)
    result: Optional[tak.Color] = None

    @property
    def logits(self):
        logits = torch.zeros((len(self.moves), encoding.MAX_MOVE_ID))
        size = self.positions[0].size
        for i in range(logits.size(0)):
            for (j, mid) in enumerate(self.moves[i]):
                logits[i, encoding.encode_move(size, mid)] = float(self.probs[i][j])
        return logits

    @property
    def results(self):
        if self.result is None:
            return [0] * len(self.positions)
        return [1.0 if p.to_move() == self.result else -1.0 for p in self.positions]


def self_play(engine, size=3):
    p = tak.Position.from_config(tak.Config(size=size))

    log = Transcript()

    tree = mcts.Node(position=p, move=None)

    while True:
        if abs(tree.v_zero) >= RESIGNATION_THRESHOLD:
            if tree.v_zero >= RESIGNATION_THRESHOLD:
                log.result = tree.position.to_move()
            else:
                log.result = tree.position.to_move().flip()
            break

        color, over = tree.position.winner()
        if over is not None:
            tree.result = color
            break

        tree = engine.analyze_tree(tree)
        probs = engine.tree_probs(tree)

        log.positions.append(tree.position)
        log.moves.append([c.move for c in tree.children])
        log.probs.append(probs.numpy())
        log.values.append(tree.value / tree.simulations)

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
    torch.manual_seed(id)
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

    while True:
        if not job.sema.acquire(block=False):
            break
        log = self_play(engine, job.size)
        job.queue.put(log)


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
    parser.add_argument("--write_games", type=str, metavar="FILE")

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

    logs = []
    with tqdm.tqdm(total=args.games) as pbar:
        while len(logs) < args.games:
            try:
                log = job.queue.get(block=True, timeout=1)
                logs.append(log)
                pbar.update()
            except queue.Empty:
                for p in processes:
                    if p.exitcode not in [0, None]:
                        raise RuntimeError("Process crashed!")

    job.shutdown.set()

    for p in processes:
        p.join()
    end = time.time()

    print(
        f"done games={len(logs)} plies={sum(len(l.positions) for l in logs)} threads={args.threads} duration={end-start:.2f} games/s={args.games/(end-start):.1f}"
    )

    if args.write_games:
        all_positions = [p for tr in logs for p in tr.positions]
        all_values = [v for tr in logs for v in tr.values]
        all_move_probs = torch.cat([tr.logits for tr in logs])
        all_results = [r for tr in logs for r in tr.results]
        encoded, mask = encoding.encode_batch(all_positions)
        torch.save(
            dict(
                positions=encoded,
                mask=mask,
                moves=all_move_probs,
                values=torch.tensor(all_values),
                results=torch.tensor(all_results),
            ),
            args.write_games,
        )

        pass


if __name__ == "__main__":
    main(sys.argv[1:])
