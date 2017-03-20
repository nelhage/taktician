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

def print_progress(epoch, session, model, test, extra=""):
  if FLAGS.evaluate:
    loss = session.run(model.mse_loss, feed_dict={
      model.x: test.positions,
      model.labels: test.results,
      model.keep_prob: 1.0
    })
    print("epoch={0} test loss={1:0.4f} {2}".format(
      epoch, loss, extra))
  else:
    loss, prec1, prec5 = session.run(
      [model.loss, model.prec1, model.prec5],
      feed_dict={
        model.x: test.positions,
        model.labels: test.moves,
        model.keep_prob: 1.0,
      })
    print("epoch={0} test loss={1:0.4f} acc={2:0.2f}%/{3:0.2f}% {4}".format(
      epoch, loss,
      100*prec1, 100*prec5,
      extra))

def main(args):
  print("Loading data...")
  train, test = tak.train.load_corpus(FLAGS.corpus,
                                      add_symmetries=FLAGS.symmetries,
                                      require_result=FLAGS.evaluate,
  )
  print("Loaded {0} training cases and {1} test cases...".format(
    len(train.positions), len(test.positions)))

  model_def = tak.proto.ModelDef(
    size    = train.size,

    layers  = FLAGS.layers,
    kernel  = FLAGS.kernel,
    filters = FLAGS.filters,
    hidden  = FLAGS.hidden,
  )
  if FLAGS.evaluate:
    model = tak.model.EvaluationModel(model_def)
  else:
    model = tak.model.PredictionModel(model_def)
  learning_rate = tf.placeholder(tf.float32)
  optimizer = getattr(tf.train, FLAGS.optimizer)(learning_rate)
  model.add_train_ops(optimizer=optimizer, regularize=FLAGS.regularize)

  if FLAGS.log_dir:
    try:
      os.makedirs(FLAGS.log_dir)
    except FileExistsError:
      pass
    with open(os.path.join(FLAGS.log_dir, "model_def"), 'wb') as fh:
      fh.write(model_def.SerializeToString())

  session = tf.InteractiveSession()
  if FLAGS.log_dir:
    writer = tf.summary.FileWriter(FLAGS.log_dir, session.graph)
    summary_op = tf.summary.merge_all()
  else:
    writer = None

  saver = tf.train.Saver(max_to_keep=10)

  if FLAGS.restore:
    saver.restore(session, FLAGS.restore)
  else:
    tf.global_variables_initializer().run()

  t_end = 0
  t_start = 0
  lr = FLAGS.eta
  if FLAGS.log_dir:
    checkpoint = os.path.join(FLAGS.log_dir, 'checkpoint')
  for epoch in range(FLAGS.epochs):
    print_progress(epoch, session, model, test, "pos/s={0:.2f}".format(
      len(train.positions)/(t_end-t_start) if t_start else 0
    ))
    if FLAGS.log_dir:
      saver.save(session, checkpoint, global_step=epoch)

    t_start = time.time()
    for i, (bx, bm, br) in enumerate(train.minibatches(FLAGS.batch)):
      feed = {
        model.x: bx,
        learning_rate: lr,
        model.keep_prob: FLAGS.dropout,
      }
      if FLAGS.evaluate:
        feed[model.labels] = br
      else:
        feed[model.labels] = bm
      ops = {
        'train': model.train_step
      }
      if writer and i % 100 == 0:
        ops['summary'] = summary_op
        ops['global_step'] = model.global_step

      vals = session.run(ops, feed_dict=feed)
      if 'summary' in vals:
        writer.add_summary(vals['summary'], vals['global_step'])

    t_end = time.time()
    if FLAGS.lr_interval and ((epoch+1) % FLAGS.lr_interval) == 0:
      lr /= FLAGS.lr_scale
      print("scaling eta={0}".format(lr))

  epoch = FLAGS.epochs
  print_progress(epoch, session, model, test, "pos/s={0:.2f}".format(
    len(train.positions)/(t_end-t_start),
  ))
  if FLAGS.log_dir:
    saver.save(session, checkpoint, global_step=epoch)

OPTIMIZERS = [
  name for name in dir(tf.train)
  if (isinstance(getattr(tf.train, name), type) and
      issubclass(getattr(tf.train, name), tf.train.Optimizer))
]

def arg_parser():
  parser = argparse.ArgumentParser()
  parser.add_argument('--corpus', type=str, default=None,
                      help='corpus to train')

  parser.add_argument('--evaluate', action='store_true',
                      default=False,
                      help='Train an evaluator instead of a classifier')

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

  parser.add_argument('--log_dir', type=str, default=None,
                      help='log directory directory')
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
