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


@define
class Transcript:
    positions: list[tak.Position] = field(factory=list)
    moves: list[list[tak.Move]] = field(factory=list)
    probs: list[np.ndarray] = field(factory=list)
    values: list[float] = field(factory=list)
    result: Optional[tak.Color] = None
    stats: mcts.Stats = field(factory=mcts.Stats)

    @property
    def logits(self):
        logits = torch.zeros((len(self.moves), encoding.MAX_MOVE_ID))
        np_view = logits.numpy()
        size = self.positions[0].size
        for i in range(logits.size(0)):
            for (j, mid) in enumerate(self.moves[i]):
                np_view[i, encoding.encode_move(size, mid)] = self.probs[i][j]
        return logits

    @property
    def results(self):
        if self.result is None:
            return [0] * len(self.positions)
        return [1.0 if p.to_move() == self.result else -1.0 for p in self.positions]


def graft_children(engine: mcts.MCTS, tree: mcts.Node) -> mcts.Node:
    if engine.config.root_noise_mix is None:
        return tree

    new_tree = mcts.Node(position=tree.position, move=tree.move)
    engine.populate(new_tree, is_root=True)

    if tree.children is None:
        return new_tree

    by_move = {n.move: n for n in tree.children}

    new_tree.simulations = tree.simulations
    new_tree.value = tree.value

    for (i, ch) in enumerate(new_tree.children):
        old_child = by_move.get(ch.move)
        if old_child is None:
            continue
        new_tree.children[i] = old_child
    return new_tree


def play_one_game(cfg, engine):
    p = tak.Position.from_config(tak.Config(size=cfg.size))

    log = Transcript()

    tree = mcts.Node(position=p, move=None)

    while True:
        if p.ply > cfg.ply_limit:
            log.result = 0.0
            break

        if abs(tree.v_zero) >= cfg.resignation_threshold:
            if tree.v_zero >= cfg.resignation_threshold:
                log.result = tree.position.to_move()
            else:
                log.result = tree.position.to_move().flip()
            break

        color, over = tree.position.winner()
        if over is not None:
            tree.result = color
            break

        tree = graft_children(engine, tree)
        tree = engine.analyze_tree(tree)
        probs = engine.tree_probs(tree)

        log.positions.append(tree.position)
        log.moves.append([c.move for c in tree.children])
        log.probs.append(probs.numpy())
        log.values.append(tree.value / tree.simulations)

        tree = tree.children[torch.multinomial(probs, 1).item()]

    log.stats = engine.stats
    engine.stats = mcts.Stats()
    return log


@define
class SelfPlayConfig:
    engine_factory: Callable
    size: int
    workers: int

    resignation_threshold: float = 0.95
    ply_limit: int = 100


@define
class BuildRemoteMCTS:
    config: mcts.Config

    host: str
    port: int = 5001

    def __call__(self):
        network = grpc.GRPCNetwork(host=self.host, port=self.port)

        return mcts.MCTS(self.config, network)


@define
class WorkerJob:
    config: SelfPlayConfig

    cmd: multiprocessing.Queue
    games: multiprocessing.Queue
    shutdown: multiprocessing.Event


def run_job(job: WorkerJob, id: int):
    engine = job.config.engine_factory()

    while True:
        id = job.cmd.get(block=True)
        if id is None:
            break
        log = play_one_game(job.config, engine)
        job.games.put(log)


def entrypoint(job: WorkerJob, id: int):
    torch.manual_seed(secrets.randbits(64))
    try:
        run_job(job, id)
        job.games.close()
        job.games.join_thread()
        job.shutdown.wait()
    except Exception as ex:
        print(f"[{id}] Process crashed: {ex}", file=sys.stderr)
        traceback.print_exc(file=sys.stderr)


@define
class MultiprocessSelfPlayEngine:
    config: SelfPlayConfig

    job: WorkerJob = field(init=False)
    next_id: int = field(default=0, init=False)
    processes: list[multiprocessing.Process] = field(factory=list, init=False)

    def __attrs_post_init__(self):
        mp = multiprocessing.get_context("spawn")
        self.job = WorkerJob(
            config=self.config,
            cmd=mp.Queue(maxsize=2 * self.config.workers),
            games=mp.Queue(maxsize=self.config.workers),
            shutdown=mp.Event(),
        )
        self.processes = [
            mp.Process(
                target=entrypoint, args=(self.job, i), name=f"selfplay-worker-{i}"
            )
            for i in range(self.config.workers)
        ]
        for p in self.processes:
            p.start()

    def play_many(self, games: int, progress: bool = False) -> list[Transcript]:
        logs = []
        todo = games
        try:
            with tqdm.tqdm(total=games, disable=not progress) as pbar:
                while len(logs) < games:
                    while todo > 0:
                        try:
                            self.job.cmd.put(self.next_id, block=False)
                            self.next_id += 1
                            todo -= 1
                        except queue.Full:
                            break

                    try:
                        log = self.job.games.get(block=True, timeout=1)
                        logs.append(log)
                        pbar.update()
                    except queue.Empty:
                        for p in self.processes:
                            if p.exitcode not in [0, None]:
                                raise RuntimeError("Process crashed!")
        except Exception:
            for p in self.processes:
                p.kill()
            raise

        return logs

    def stop(self):
        for _ in range(self.config.workers):
            self.job.cmd.put(None, block=False)

        self.job.shutdown.set()
        for p in self.processes:
            p.join()


def play_many_games(
    config: SelfPlayConfig, games: int, progress: bool = False
) -> list[Transcript]:
    engine = MultiprocessSelfPlayEngine(config=config)

    try:
        return engine.play_many(games, progress=progress)
    finally:
        engine.stop()


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
        results=torch.tensor(all_results, dtype=torch.float32),
    )
