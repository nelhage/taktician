import tak
import tak.ptn
import tak.train

import argparse

def main(args):
  model = tak.train.load_model(FLAGS.model, eval_symmetries=FLAGS.symmetries)
  pos = tak.Position.from_config(tak.Config(size=5))

  with open(FLAGS.out, 'w') as ptn:
    ptn.write('[Size "{0}"]\n'.format(pos.size))
    while True:
      if pos.ply % 2 == 0:
        ptn.write("\n{0}. ".format(1+int(pos.ply/2)))
      if pos.ply == 0:
        m = tak.ptn.parse_move('a1')
        pos = pos.move(m)
      elif pos.ply == 1:
        m = tak.ptn.parse_move('a5')
        pos = pos.move(m)
      else:
        m, pos = model.get_move(pos)
      ptn.write(tak.ptn.format_move(m))
      ptn.write(" ")
      who, why = pos.winner()
      if why is not None:
        break
    ptn.write("\n")

def arg_parser():
  parser = argparse.ArgumentParser()
  parser.add_argument('--model', type=str, default=None,
                      help='model to run')

  parser.add_argument('--symmetries',
                      default=False,
                      action='store_true',
                      help='average over all symmetries')

  parser.add_argument('--out', type=str, default='game.ptn')
  return parser

if __name__ == '__main__':
  parser = arg_parser()
  FLAGS, unparsed = parser.parse_known_args()
  main(unparsed)
