import secrets
import sys
import traceback
from typing import Optional, Callable

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


def play_one_game(engine, size=3):
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
class SelfPlayConfig:
    engine_factory: Callable
    size: int
    games: int
    workers: int


@define
class BuildRemoteMCTS:
    simulations: int

    host: str
    port: int = 5001

    def __call__(self):
        network = grpc.GRPCNetwork(host=self.host, port=self.port)

        return mcts.MCTS(
            mcts.Config(
                network=network,
                simulation_limit=self.simulations,
                time_limit=0,
            )
        )


@define
class WorkerJob:
    config: SelfPlayConfig

    sema: multiprocessing.Semaphore
    queue: multiprocessing.Queue
    shutdown: multiprocessing.Event


def run_job(job: WorkerJob, id: int):
    engine = job.config.engine_factory()

    while True:
        if not job.sema.acquire(block=False):
            break
        log = play_one_game(engine, job.config.size)
        job.queue.put(log)


def entrypoint(job: WorkerJob, id: int):
    torch.manual_seed(secrets.randbits(64))
    try:
        run_job(job, id)
        job.queue.close()
        job.queue.join_thread()
        job.shutdown.wait()
    except Exception as ex:
        print(f"[{id}] Process crashed: {ex}", file=sys.stderr)
        traceback.print_exc(file=sys.stderr)


def play_many_games(config: SelfPlayConfig, progress: bool = False) -> list[Transcript]:
    job = WorkerJob(
        config=config,
        sema=multiprocessing.Semaphore(value=config.games),
        queue=multiprocessing.Queue(maxsize=config.workers),
        shutdown=multiprocessing.Event(),
    )

    processes = [
        multiprocessing.Process(
            target=entrypoint, args=(job, i), name=f"selfplay-worker-{i}"
        )
        for i in range(config.workers)
    ]
    for p in processes:
        p.start()

    logs = []
    try:
        with tqdm.tqdm(total=config.games, disable=not progress) as pbar:
            while len(logs) < config.games:
                try:
                    log = job.queue.get(block=True, timeout=1)
                    logs.append(log)
                    pbar.update()
                except queue.Empty:
                    for p in processes:
                        if p.exitcode not in [0, None]:
                            raise RuntimeError("Process crashed!")
    except Exception:
        for p in processes:
            p.kill()
        raise

    job.shutdown.set()

    for p in processes:
        p.join()

    return logs


def encode_games(logs: list[Transcript]):
    all_positions = [p for tr in logs for p in tr.positions]
    all_values = [v for tr in logs for v in tr.values]
    all_move_probs = torch.cat([tr.logits for tr in logs])
    all_results = [r for tr in logs for r in tr.results]
    encoded, mask = encoding.encode_batch(all_positions)
    return dict(
        positions=encoded,
        mask=mask,
        moves=all_move_probs,
        values=torch.tensor(all_values),
        results=torch.tensor(all_results),
    )
