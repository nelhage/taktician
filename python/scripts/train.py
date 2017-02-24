import tak.ptn
import tak.train

import sys
import os
import argparse
import csv

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
      self.y_ = tf.placeholder(tf.float32, (None, mcount))

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

    with tf.name_scope('Output'):
      self.keep_prob = tf.placeholder(tf.float32)
      icount = size*size*FLAGS.filters
      self.W = tf.Variable(tf.zeros([icount, mcount]))
      self.b = tf.Variable(tf.zeros([mcount]))

      x = tf.reshape(tf.nn.dropout(activations, keep_prob=self.keep_prob),
                     [-1, icount])
      self.y = tf.matmul(x, self.W) + self.b

    with tf.name_scope('Train'):
      self.cross_entropy = tf.reduce_mean(
        tf.nn.softmax_cross_entropy_with_logits(logits=self.y, labels=self.y_))
      self.regularization_loss = tf.contrib.layers.apply_regularization(
        tf.contrib.layers.l2_regularizer(FLAGS.regularize),
      )

      self.loss = self.cross_entropy + self.regularization_loss
      self.train_step = (tf.train.GradientDescentOptimizer(FLAGS.eta).
                         minimize(self.loss))

      correct = tf.equal(tf.argmax(self.y, 1), tf.argmax(self.y_, 1))
      self.accuracy = tf.reduce_mean(tf.cast(correct, tf.float32))

def main(args):
  print("Loading data...")
  train, test = tak.train.load_corpus(FLAGS.corpus)
  print("Loaded {0} training cases and {1} test cases...".format(
    len(train.positions), len(test.positions)))

  model = TakModel(train.size)

  sess = tf.InteractiveSession()

  tf.global_variables_initializer().run()

  for epoch in range(FLAGS.epochs):
    loss, acc = sess.run([model.loss, model.accuracy],
                         feed_dict={
                           model.x: test.positions,
                           model.y_: test.moves,
                           model.keep_prob: 1.0,
                         })
    print("epoch={0} test loss={1:0.4f} acc={2:0.2f}%".format(
      epoch, loss, 100*acc))

    for (bx, by) in train.minibatches(FLAGS.batch):
      sess.run(model.train_step, feed_dict={
        model.x: bx,
        model.y_: by,
        model.keep_prob: FLAGS.dropout,
      })

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

  parser.add_argument('--eta', type=float, default=0.5,
                      help='learning rate')
  parser.add_argument('--regularize', type=float, default=1e-6,
                      help='L2 regularization scale')
  parser.add_argument('--dropout', type=float, default=0.5,
                      help='L2 regularization scale')
  parser.add_argument('--batch', type=int, default=100,
                      help='batch size')
  parser.add_argument('--epochs', type=int, default=30,
                      help='epochs')
  return parser

if __name__ == '__main__':
  parser = arg_parser()
  FLAGS, unparsed = parser.parse_known_args()
  tf.app.run(main=main, argv=[sys.argv[0]] + unparsed)
else:
  FLAGS, _ = arg_parser().parse_args([])
