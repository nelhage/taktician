import tak.ptn
import tak.train

import sys
import os
import argparse
import csv

import attr

import numpy as np
import tensorflow as tf

FLAGS = None

@attr.s(frozen=True)
class Dataset(object):
  size = attr.ib()
  positions = attr.ib()
  moves = attr.ib()

def load_corpus_file(path):
  positions = []

  with open(path) as f:
    reader = csv.reader(f)
    for row in reader:
      tps, m = row[:2]
      positions.append((
        tak.ptn.parse_tps(tps),
        tak.ptn.parse_move(m)))

  size = positions[0][0].size

  xs = np.zeros((len(positions),) + tak.train.feature_shape(size))
  ys = np.zeros((len(positions), tak.train.move_count(size)))

  for i, (p, m) in enumerate(positions):
    tak.train.features(p, xs[i])
    ys[i][tak.train.move2id(m, size)] = 1
  return Dataset(size, xs, ys)

def load_corpus(dir):
  return (
    load_corpus_file(os.path.join(dir, 'train.csv')),
    load_corpus_file(os.path.join(dir, 'test.csv')))

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

    with tf.name_scope('Softmax'):
      self.W = tf.Variable(tf.zeros([fcount, mcount]))
      self.b = tf.Variable(tf.zeros([mcount]))
      x = tf.reshape(self.x, [-1, fcount])
      self.y = tf.matmul(x, self.W) + self.b

    with tf.name_scope('Train'):
      self.cross_entropy = tf.reduce_mean(
        tf.nn.softmax_cross_entropy_with_logits(logits=self.y, labels=self.y_))
      self.train_step = (tf.train.GradientDescentOptimizer(FLAGS.eta).
                         minimize(self.cross_entropy))

      self.loss = self.cross_entropy

      correct = tf.equal(tf.argmax(self.y, 1), tf.argmax(self.y_, 1))
      self.accuracy = tf.reduce_mean(tf.cast(correct, tf.float32))


def batch(dataset, batch):
  perm = np.random.permutation(len(dataset.positions))
  i = 0
  while i < len(dataset.positions):
    yield (dataset.positions[perm[i:i+batch]], dataset.moves[perm[i:i+batch]])
    i += batch

def main(args):
  print("Loading data...")
  train, test = load_corpus(FLAGS.corpus)
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
                         })
    print("epoch={0} test loss={1:0.4f} acc={2:0.2f}%".format(
      epoch, loss, 100*acc))

    for (bx, by) in batch(train, FLAGS.batch):
      sess.run(model.train_step, feed_dict={
        model.x: bx,
        model.y_: by,
      })

if __name__ == '__main__':
  parser = argparse.ArgumentParser()
  parser.add_argument('--corpus', type=str, default=None,
                      help='corpus to train')

  parser.add_argument('--eta', type=float, default=0.5,
                      help='learning rate')
  parser.add_argument('--batch', type=int, default=100,
                      help='batch size')
  parser.add_argument('--epochs', type=int, default=30,
                      help='epochs')

  FLAGS, unparsed = parser.parse_known_args()
  tf.app.run(main=main, argv=[sys.argv[0]] + unparsed)
