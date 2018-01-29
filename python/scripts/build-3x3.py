import tak.ptn
import tak.game
import tak.train
import tak.proto
import tak.model
import tak.symmetry

import argparse
import grpc
import attr
import random

FLAGS = None

def terminal_value(value):
   return value > (1<<29) or value < -(1<<29)

class Node(object):
  def __init__(self, position):
    self.position = position
    self.pv = None
    self.value = None
    self.terminal = position.winner()[0] is not None
    self.children = {}

def select_child(all_moves, root):
  variant = []
  node = root
  while node.pv is not None:
    while True:
      if node.pv and not node.terminal and FLAGS.greedy and random.random() < FLAGS.greedy:
        mv = node.pv
      else:
        mv = random.choice(all_moves)
      if mv not in node.children:
        try:
          pos = node.position.move(mv)
        except tak.game.IllegalMove:
          continue
        node.children[mv] = Node(pos)
        break
      node = node.children[mv]
      variant.append(mv)
      if node.terminal:
        variant = []
        node = root

  return (variant, node)

def store_pvs(child, pvs, value):
  if not terminal_value(value):
    pvs = pvs[:1]

  for pv in pvs:
    child.pv = tak.ptn.parse_move(pv)
    child.value = value
    if terminal_value(value):
      child.terminal = True
    try:
      pos = child.position.move(child.pv)
    except tak.game.IllegalMove:
      return
    child.children[child.pv] = Node(pos)

    child = child.children[child.pv]
    value = -value

def collect_children(root):
  stk = [root]
  while stk:
    node = stk.pop()
    if node.pv is not None:
      yield tak.proto.CorpusEntry(
        tps=tak.ptn.format_tps(node.position),
        move=tak.ptn.format_move(node.pv)
      )
    stk.extend(node.children.values())

def main(args):
  channel = grpc.insecure_channel(FLAGS.server)
  stub = tak.proto.TakticianStub(channel)

  root = Node(tak.game.Position.from_config(tak.game.Config(size=3)))
  all_moves = tak.enumerate_moves(3)

  for i in range(FLAGS.iterations):
    variant, child = select_child(all_moves, root)
    resp = stub.Analyze(
      tak.proto.AnalyzeRequest(position=tak.ptn.format_tps(child.position),
                               depth=8))
    print("variant={} pv={} v={}".format(
      [tak.ptn.format_move(m) for m in variant],
      list(resp.pv), resp.value))
    store_pvs(child, list(resp.pv), resp.value)
  if FLAGS.output is not None:
    tak.train.corpus.write_proto(FLAGS.output, collect_children(root))

def arg_parser():
  parser = argparse.ArgumentParser()
  parser.add_argument('--server', type=str, default='localhost:55430',
                      help='taktician server')
  parser.add_argument('--iterations', type=int, default=100,
                      help='examine N positions')
  parser.add_argument('--output', type=str, default=None,
                      help='write corpus to file')
  parser.add_argument('--greedy', type=float, default=None,
                      help='explore pv with probability')

  return parser

if __name__ == '__main__':
  parser = arg_parser()
  FLAGS, unparsed = parser.parse_known_args()
  main(unparsed)
else:
  FLAGS, _ = arg_parser().parse_args([])
