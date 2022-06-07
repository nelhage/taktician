from dataclasses import dataclass
import math
import random
import typing as T
from . import game, ptn, moves
import time


@dataclass
class Config:
    time_limit: float = 1.0
    simulation_limit: int = 0
    max_rollout: int = 1_000_000
    C: float = 0.7
    seed: T.Optional[int] = None

    evaluate: T.Optional[T.Callable[[game.Position], float]] = None


@dataclass
class Node:
    position: game.Position
    move: T.Optional[moves.Move]

    simulations: int = 0
    value: float = 0

    children: T.Optional[list["Node"]] = None

    def ucb(self, C: float, N: int):
        if self.simulations == 0:
            return 10
        return -(self.value / self.simulations) + C * math.sqrt(
            math.log(N) / self.simulations
        )


def extract_pv(node):
    pv = []
    while node.children is not None:
        best = max(node.children, key=lambda n: n.simulations)
        pv.append(best)
        node = best
    return pv


Elem = T.TypeVar("Elem")
Key = T.TypeVar("Key")


def max_tiebreak(
    seq: T.Iterable[Elem], key: T.Callable[[Elem], Key], random=random
) -> Elem:
    i = 0
    seq = iter(seq)
    best = next(seq)
    best_key = key(best)
    for elem in seq:
        elem_key = key(elem)
        if elem_key > best_key:
            best = elem
            best_key = elem_key
            i = 1
        elif elem_key == best_key:
            if random.randint(0, i) == 0:
                best = elem
            i += 1
    return best


class MCTS:
    def __init__(self, config: Config):
        self.config = config
        self.random = random.Random(config.seed)

    def analyze(self, p: game.Position) -> Node:
        tree = Node(
            position=p,
            move=None,
        )

        start = time.monotonic()
        if self.config.time_limit > 0:
            deadline = start + self.config.time_limit
        else:
            deadline = float("inf")
        simulation_limit = self.config.simulation_limit

        while True:
            if time.monotonic() > deadline or (
                simulation_limit > 0 and tree.simulations >= simulation_limit
            ):
                break

            path = self.descend(tree)
            self.populate(path[-1])
            value = self.evaluate(path[-1])
            self.update(path, value)

        return tree

    def print_tree(self, tree: Node):
        for child in sorted(tree.children, key=lambda c: -c.simulations):
            print(
                f"{ptn.format_move(child.move):>4}"
                f" visit={child.simulations:>3d}"
                f" value={-child.value/child.simulations:+5.2f}"
                f" ucb={child.ucb(self.config.C, tree.simulations):0.3f}"
            )

    def get_move(self, p: game.Position) -> moves.Move:
        tree = self.analyze(p)
        return self.select_root_move(tree)

    def descend(self, tree: Node) -> list[Node]:
        path = []
        while True:
            path.append(tree)
            if tree.children is None:
                return path

            best = max_tiebreak(
                tree.children,
                key=lambda n: n.ucb(self.config.C, tree.simulations),
                random=self.random,
            )
            tree = best

    def populate(self, node: Node):
        moves = node.position.all_moves()
        node.children = []

        for m in moves:
            try:
                child = node.position.move(m)
            except game.IllegalMove:
                continue
            color, _ = child.winner()

            child_node = Node(
                position=child,
                move=m,
            )

            if color is not None:
                if color == child.to_move():
                    child_node.value = 1
                else:
                    child_node.value = -1
            node.children.append(child_node)

    def evaluate(self, node: Node) -> float:
        if self.config.evaluate is not None:
            return self.config.evaluate(node)
        return self.rollout(node)

    def rollout(self, node: Node) -> float:
        i = 0
        pos = node.position
        winner = None
        while i < self.config.max_rollout:
            (winner, _) = pos.winner()
            if winner is not None:
                break

            (move, pos) = self.select_rollout_move(pos)

        if winner is not None:
            if winner == node.position.to_move():
                return 1.0
            else:
                return -1.0
        # todo: scoring heuristic
        return 0.0

    def select_rollout_move(self, pos: game.Position):
        moves = pos.all_moves()
        while True:
            m = self.random.choice(moves)
            try:
                child = pos.move(m)
                return (m, child)
            except game.IllegalMove:
                continue

    def update(self, path: list[Node], val: float):
        for node in reversed(path):
            node.value += val
            node.simulations += 1
            val = -val

    def select_root_move(self, tree: Node) -> moves.Move:
        return max_tiebreak(
            tree.children, key=lambda c: c.simulations, random=self.random
        ).move
