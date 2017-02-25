import tak.ptn
import tak.train

import argparse
import csv
import os
import sys
import time

import numpy as np
import tensorflow as tf

FLAGS = None

class TakModel(object):
  def __init__(self, size):
    self.size = size

    fshape = tak.train.feature_shape(size)
    _, _, fplanes = fshape
    fcount = fplanes * size * size
    mcount = tak.train.move_count(size)

    with tf.name_scope('Input'):
      self.x = tf.placeholder(tf.float32, (None,) + fshape)
      self.labels = tf.placeholder(tf.float32, (None, mcount))

    with tf.name_scope('Hidden'):
      activations = self.x
      self.layers = []
      for i in range(FLAGS.layers):
        with tf.name_scope('Layer{0}'.format(i)):
          activations = tf.contrib.layers.convolution2d(
            activations,
            num_outputs=FLAGS.filters,
            padding='SAME',
            kernel_size=FLAGS.kernel,
            trainable=True,
            variables_collections={'weights': [tf.GraphKeys.WEIGHTS]},
          )
          self.layers.append(activations)

      icount = size*size*FLAGS.filters
      if FLAGS.hidden > 0:
        self.W_h = tf.Variable(tf.zeros([icount, FLAGS.hidden]), name="weights")
        self.b_h = tf.Variable(tf.zeros([FLAGS.hidden]), name="biases")

        x = tf.reshape(activations, [-1, icount])
        activations = tf.nn.relu(tf.matmul(x, self.W_h) + self.b_h)
        icount = FLAGS.hidden

    with tf.name_scope('Output'):
      self.keep_prob = tf.placeholder_with_default(
        tf.ones(()), shape=(), name='keep_prob')
      self.W = tf.Variable(tf.zeros([icount, mcount]), name="weights")
      self.b = tf.Variable(tf.zeros([mcount]), name="biases")

      x = tf.reshape(tf.nn.dropout(activations, keep_prob=self.keep_prob),
                     [-1, icount])
      self.logits = tf.matmul(x, self.W) + self.b

    with tf.name_scope('Train'):
      self.cross_entropy = tf.reduce_mean(
        tf.nn.softmax_cross_entropy_with_logits(logits=self.logits, labels=self.labels))
      self.regularization_loss = tf.contrib.layers.apply_regularization(
        tf.contrib.layers.l2_regularizer(FLAGS.regularize),
      )

      self.loss = self.cross_entropy + self.regularization_loss
      self.global_step = tf.Variable(0, name='global_step', trainable=False)
      self.train_step = (tf.train.GradientDescentOptimizer(FLAGS.eta).
                         minimize(self.loss, global_step=self.global_step))

      correct = tf.equal(tf.argmax(self.labels, 1), tf.argmax(self.logits, 1))
      self.accuracy = tf.reduce_mean(tf.cast(correct, tf.float32))

    tf.add_to_collection('inputs', self.x)
    tf.add_to_collection('logits', self.logits)

def main(args):
  print("Loading data...")
  train, test = tak.train.load_corpus(FLAGS.corpus)
  print("Loaded {0} training cases and {1} test cases...".format(
    len(train.positions), len(test.positions)))

  model = TakModel(train.size)

  session = tf.InteractiveSession()
  saver = tf.train.Saver(max_to_keep=10)

  if FLAGS.restore:
    saver.restore(session, FLAGS.restore)
  else:
    tf.global_variables_initializer().run()

  t_end = 0
  t_start = 0
  for epoch in range(FLAGS.epochs):
    loss, acc = session.run([model.loss, model.accuracy],
                         feed_dict={
                           model.x: test.positions,
                           model.labels: test.moves,
                           model.keep_prob: 1.0,
                         })
    print("epoch={0} test loss={1:0.4f} acc={2:0.2f}% pos/s={3:.2f}".format(
      epoch, loss, 100*acc, len(train.positions)/(t_end-t_start) if t_start else 0))
    if FLAGS.checkpoint:
      saver.save(session, FLAGS.checkpoint, global_step=epoch)

    t_start = time.time()
    for (bx, by) in train.minibatches(FLAGS.batch):
      session.run(model.train_step, feed_dict={
        model.x: bx,
        model.labels: by,
        model.keep_prob: FLAGS.dropout,
      })
    t_end = time.time()

  if FLAGS.write_metagraph:
    tf.train.export_meta_graph(filename=FLAGS.write_metagraph)

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

  parser.add_argument('--eta', type=float, default=0.5,
                      help='learning rate')
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
  parser.add_argument('--write-metagraph', type=str, default=None,
                      help='write metagraph to path')
  return parser

if __name__ == '__main__':
  parser = arg_parser()
  FLAGS, unparsed = parser.parse_known_args()
  tf.app.run(main=main, argv=[sys.argv[0]] + unparsed)
else:
  FLAGS, _ = arg_parser().parse_args([])
