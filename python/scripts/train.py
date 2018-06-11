import tak.ptn
import tak.train
import tak.proto
import tak.model

import csv
import os
import sys
import time

import numpy as np
import tensorflow as tf

tf.flags.DEFINE_string('corpus', default=None, help='corpus to train')

tf.flags.DEFINE_integer('kernel', default=3, help='convolutional kernel size')
tf.flags.DEFINE_integer('filters', default=16, help='convolutional filters')
tf.flags.DEFINE_integer('layers', default=2, help='number of convolutional layers')
tf.flags.DEFINE_integer('hidden', default=0, help='number of hidden fully-connected nodes')

tf.flags.DEFINE_string('optimizer',
                       default='GradientDescentOptimizer',
                       help='tensorflow optimizer class')
                       # choices=OPTIMIZERS)
tf.flags.DEFINE_float('eta', default=0.05, help='learning rate')
tf.flags.DEFINE_integer('lr_scale', default=1, help='scale learning rate down')
tf.flags.DEFINE_integer('lr_interval', default=None, help='scale learning rate every N epochs')
tf.flags.DEFINE_float('regularize', default=1e-6, help='L2 regularization scale')
tf.flags.DEFINE_float('dropout', default=0.5, help='dropout fraction')
tf.flags.DEFINE_integer('batch', default=100, help='batch size')
tf.flags.DEFINE_integer('epochs', default=30, help='train epochs')

tf.flags.DEFINE_string('checkpoint', default=None, help='checkpoint directory')
tf.flags.DEFINE_string('restore', default=None, help='restore from path')

tf.flags.DEFINE_boolean('symmetries', default=True, help='Add symmetries to corpus')

FLAGS = tf.flags.FLAGS

def main(args):
  print("Loading data...")
  t = time.time()
  train, test = tak.train.load_features(FLAGS.corpus, add_symmetries=FLAGS.symmetries)
  e = time.time()
  print("Loaded {0} training cases and {1} test cases in {2:.3f}s...".format(
    len(train), len(test), e-t))

  model_def = tak.proto.ModelDef(
    size    = train.size,

    layers  = FLAGS.layers,
    kernel  = FLAGS.kernel,
    filters = FLAGS.filters,
    hidden  = FLAGS.hidden,
  )
  model = tak.model.PredictionModel(model_def)
  learning_rate = tf.placeholder(tf.float32)
  optimizer = getattr(tf.train, FLAGS.optimizer)(learning_rate)
  model.add_train_ops(optimizer=optimizer, regularize=FLAGS.regularize)

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
        model.x: test.instances[0],
        model.labels: test.instances[1],
        model.keep_prob: 1.0,
      })
    print("epoch={0} test loss={1:0.4f} acc={2:0.2f}%/{3:0.2f}% pos/s={4:.2f}".format(
      epoch, loss,
      100*prec1, 100*prec5,
      len(train)/(t_end-t_start) if t_start else 0))
    if FLAGS.checkpoint:
      saver.save(session, FLAGS.checkpoint, global_step=epoch)

    t_start = time.time()
    for (bx, by) in train.minibatches(FLAGS.batch):
      session.run(model.train_step, feed_dict={
        model.x: bx,
        model.labels: by,
        learning_rate: lr,
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

if __name__ == '__main__':
  tf.app.run(main=main, argv=sys.argv)
