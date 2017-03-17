import tak.ptn
import tak.train
import tak.proto
import tak.model

import argparse
import csv
import os
import sys
import time

import numpy as np
import tensorflow as tf

FLAGS = None

def main(args):
  print("Loading data...")
  train, test = tak.train.load_corpus(FLAGS.corpus, add_symmetries=FLAGS.symmetries)
  print("Loaded {0} training cases and {1} test cases...".format(
    len(train.positions), len(test.positions)))

  model_def = tak.proto.ModelDef(
    size    = train.size,

    layers  = FLAGS.layers,
    kernel  = FLAGS.kernel,
    filters = FLAGS.filters,
    hidden  = FLAGS.hidden,
  )
  model = tak.model.PredictionModel(model_def)
  model.add_train_ops(FLAGS.regularize,
                      optimizer=FLAGS.optimizer)

  if FLAGS.checkpoint:
    with open(FLAGS.checkpoint + ".model", 'wb') as fh:
      fh.write(model_def.SerializeToString())

  session = tf.InteractiveSession()
  saver = tf.train.Saver(max_to_keep=10)

  if FLAGS.restore:
    saver.restore(session, FLAGS.restore)
  else:
    tf.global_variables_initializer().run()

  t_end = 0
  t_start = 0
  lr = FLAGS.eta
  for epoch in range(FLAGS.epochs):
    loss, prec1, prec5 = session.run(
      [model.loss, model.prec1, model.prec5],
      feed_dict={
        model.x: test.positions,
        model.labels: test.moves,
        model.keep_prob: 1.0,
      })
    print("epoch={0} test loss={1:0.4f} acc={2:0.2f}%/{3:0.2f}% pos/s={4:.2f}".format(
      epoch, loss,
      100*prec1, 100*prec5,
      len(train.positions)/(t_end-t_start) if t_start else 0))
    if FLAGS.checkpoint:
      saver.save(session, FLAGS.checkpoint, global_step=epoch)

    t_start = time.time()
    for (bx, by) in train.minibatches(FLAGS.batch):
      session.run(model.train_step, feed_dict={
        model.x: bx,
        model.labels: by,
        model.learning_rate: lr,
        model.keep_prob: FLAGS.dropout,
      })
    t_end = time.time()
    if FLAGS.lr_interval and ((epoch+1) % FLAGS.lr_interval) == 0:
      lr /= FLAGS.lr_scale
      print("scaling eta={0}".format(lr))

OPTIMIZERS = [
  name for name in dir(tf.train)
  if (isinstance(getattr(tf.train, name), type) and
      issubclass(getattr(tf.train, name), tf.train.Optimizer))
]

def arg_parser():
  parser = argparse.ArgumentParser()
  parser.add_argument('--corpus', type=str, default=None,
                      help='corpus to train')

  parser.add_argument('--kernel', type=int, default=3,
                      help='convolutional kernel size')
  parser.add_argument('--filters', type=int, default=16,
                      help='convolutional filters')
  parser.add_argument('--layers', type=int, default=2,
                      help='number of convolutional layers')
  parser.add_argument('--hidden', type=int, default=0,
                      help='number of hidden fully-connected nodes')

  parser.add_argument('--optimizer',
                      type=str,
                      default='GradientDescentOptimizer',
                      help='tensorflow optimizer class',
                      choices=OPTIMIZERS)
  parser.add_argument('--eta', type=float, default=0.05,
                      help='learning rate')
  parser.add_argument('--lr_scale', type=int, default=1,
                      help='scale learning rate down')
  parser.add_argument('--lr_interval', type=int, default=None,
                      help='scale learning rate every N epochs')
  parser.add_argument('--regularize', type=float, default=1e-6,
                      help='L2 regularization scale')
  parser.add_argument('--dropout', type=float, default=0.5,
                      help='L2 regularization scale')
  parser.add_argument('--batch', type=int, default=100,
                      help='batch size')
  parser.add_argument('--epochs', type=int, default=30,
                      help='train epochs')

  parser.add_argument('--checkpoint', type=str, default=None,
                      help='checkpoint directory')
  parser.add_argument('--restore', type=str, default=None,
                      help='restore from path')

  parser.add_argument('--no-symmetries', default=True,
                      dest='symmetries', action='store_false')
  return parser

if __name__ == '__main__':
  parser = arg_parser()
  FLAGS, unparsed = parser.parse_known_args()
  tf.app.run(main=main, argv=unparsed)
else:
  FLAGS, _ = arg_parser().parse_args([])
