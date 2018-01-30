import tak.ptn
import tak.game
import tak.train
import tak.proto
import tak.model
import tak.symmetry

import datetime
import argparse
import grpc
import attr
import random
import os

FLAGS = None

def terminal_value(value):
   return value > (1<<29) or value < -(1<<29)

def play_rollout(stub, all_moves, date, id):
  ms = []
  g = tak.game.Position.from_config(tak.game.Config(size=3))
  g = g.move(tak.ptn.parse_move('a1'))

  if random.random() < 0.5:
    g = g.move(tak.ptn.parse_move('a3'))
  else:
    g = g.move(tak.ptn.parse_move('c3'))

  while not g.winner()[0]:
    resp = stub.Analyze(
      tak.proto.AnalyzeRequest(position=tak.ptn.format_tps(g), depth=8))
    greedy = FLAGS.greedy and random.random() < FLAGS.greedy

    while True:
      if greedy:
        mv = tak.ptn.parse_move(resp.pv[0])
      else:
        mv = random.choice(all_moves)

      try:
        nextg = g.move(mv)
        break
      except tak.game.IllegalMove as ex:
        if greedy:
          print("wtf tako; illegal move b='{}' mv={} e={}".format(
            tak.ptn.format_tps(g),
            tak.ptn.format_move(mv),
            ex,
          ))
          import pdb; pdb.set_trace()
        continue

    ms.append(tak.proto.CorpusEntry(
      day = date,
      id = id,
      ply = g.ply,
      tps = tak.ptn.format_tps(g),
      move = resp.pv[0],
      value = resp.value,
    ))
    g = nextg
  return ms

def flatten_rollouts(rollouts):
  for game in rollouts:
    for pos in game:
      yield pos

def main(args):
  channel = grpc.insecure_channel(FLAGS.server)
  stub = tak.proto.TakticianStub(channel)

  all_moves = tak.enumerate_moves(3)

  rollouts = []

  date = str(datetime.date.today())
  for i in range(FLAGS.iterations):
    rollout = play_rollout(stub, all_moves, date, i)
    rollouts.append(rollout)
    print("pvs={}".format(
      [m.move for m in rollout]))

  if FLAGS.output is not None:
    try:
      os.makedirs(FLAGS.output)
    except FileExistsError:
      pass
    train, test = [], []
    for entry in flatten_rollouts(rollouts):
      if random.random() < FLAGS.test_fraction:
        test.append(entry)
      else:
        train.append(entry)
      tak.train.corpus.write_proto(os.path.join(FLAGS.output, 'train.dat'), train)
      tak.train.corpus.write_proto(os.path.join(FLAGS.output, 'test.dat'), test)

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
  parser.add_argument('--test-fraction', default=0.05, type=float,
                      help='select fraction of games to use for training set')

  return parser

if __name__ == '__main__':
  parser = arg_parser()
  FLAGS, unparsed = parser.parse_known_args()
  main(unparsed)
else:
  FLAGS, _ = arg_parser().parse_args([])
