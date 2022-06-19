import math
import random
import time
import typing as T
from dataclasses import dataclass

from . import game, moves, ptn
from .model import encoding

import torch


class PolicyAndAction(T.Protocol):
    def evaluate(self, position: game.Position) -> tuple[torch.Tensor, float]:
        ...


@dataclass
class Config:
    network: PolicyAndAction

    time_limit: float = 1.0
    simulation_limit: int = 0

    C: float = 0.7
    cutoff_prob: float = 1e-6
    seed: T.Optional[int] = None


ALPHA_EPSILON = 1e-3


@dataclass
class Node:
    position: game.Position
    move: T.Optional[moves.Move]

    v_zero: float = 0
    value: float = 0
    simulations: int = 0

    child_probs: T.Optional[torch.Tensor] = None
    children: T.Optional[list["Node"]] = None

    def policy_probs(self, c: float) -> torch.Tensor:
        pi_theta = self.child_probs

        if self.simulations == 0:
            return pi_theta

        q = torch.tensor(
            [
                -c.value / c.simulations if c.simulations > 0 else self.v_zero
                for c in self.children
            ]
        )

        lambda_n = (
            c * math.sqrt(self.simulations) / (self.simulations + len(self.children))
        )

        alpha_min = (q + lambda_n * pi_theta).max().item()
        alpha_max = (q + lambda_n).max().item()
        alpha = (alpha_max + alpha_min) / 2

        iters = 0

        while True:
            iters += 1
            if iters > 32:
                raise AssertionError("search for alpha did not terminate")
            pi_alpha = lambda_n * pi_theta / (alpha - q)
            sigma = pi_alpha.sum()

            # print(f"i={iters} sigma={sigma:0.2f}")

            if (1 - sigma).abs() <= ALPHA_EPSILON or (alpha_max - alpha_min) <= 1e-6:
                return pi_alpha
            if sigma > 1:
                alpha_min = alpha
                alpha = (alpha + alpha_max) / 2
            else:
                alpha_max = alpha
                alpha = (alpha + alpha_min) / 2


# def extract_pv(node):
#     pv = []
#     while node.children is not None:
#         best = max(node.children, key=lambda n: n.simulations)
#         pv.append(best)
#         node = best
#     return pv


Elem = T.TypeVar("Elem")
Key = T.TypeVar("Key")


class MCTS:
    def __init__(self, config: Config):
        self.config = config
        self.random = random.Random(config.seed)

    def analyze(self, p: game.Position) -> Node:
        tree = Node(position=p, move=None)
        self.root = tree

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
            self.update(path)

        return tree

    def print_tree(self, tree: Node):
        print(
            f"node visits={tree.simulations} v0={tree.v_zero:+0.2f} value={tree.value/tree.simulations:+0.2f}"
        )
        policy = tree.policy_probs(self.config.C)
        for i in torch.argsort(-policy).tolist():
            child = tree.children[i]
            prob = policy[i]
            if prob < 0.01:
                continue
            print(
                f"  {ptn.format_move(child.move):>4}"
                f" visit={child.simulations:>3d}"
                f" value={-child.value/child.simulations if child.simulations else child.v_zero:+5.2f}"
                f" pi_theta[a]={tree.child_probs[i]:0.2f}"
                f" pi[a]={prob:0.2f}"
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

            policy = tree.policy_probs(self.config.C)
            child = torch.multinomial(policy, 1).item()
            tree = tree.children[child]

    def populate(self, node: Node):
        winner, why = node.position.winner()
        if why is not None:
            if winner == node.position.to_move():
                node.v_zero = 1
            elif winner == node.position.to_move().flip():
                node.v_zero = -1
            else:
                node.v_zero = 0
            return

        raw_probs, node.v_zero = self.config.network.evaluate(node.position)

        sort = torch.sort(raw_probs, descending=True)
        child_probs = []
        node.children = []

        for (mid, prob) in zip(sort.indices.numpy(), sort.values.numpy()):
            if prob < self.config.cutoff_prob:
                break
            m = encoding.decode_move(node.position.size, mid)
            if m is None:
                continue
            try:
                child = node.position.move(m)
            except game.IllegalMove:
                continue

            node.children.append(Node(position=child, move=m))
            child_probs.append(prob)

        child_probs = torch.tensor(child_probs)
        child_probs /= child_probs.sum()
        node.child_probs = child_probs

    def update(self, path: list[Node]):
        value = path[-1].v_zero

        for node in reversed(path):
            node.value += value
            node.simulations += 1
            value = -value

    def select_root_move(self, tree: Node) -> moves.Move:
        policy = tree.policy_probs(self.config.C)
        child = torch.multinomial(policy, 1).item()
        return tree.children[child].move
