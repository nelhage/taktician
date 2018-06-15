import tak.ptn
import tak.train
import tak.proto
import tak.model

import os
import sys
import time

import numpy as np
import tensorflow as tf

tf.flags.DEFINE_string('corpus', default=None, help='corpus to train')
tf.flags.DEFINE_string('features', default=None, help='saved features to train')
tf.flags.DEFINE_integer('size', default=5, help='board size')

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
  if FLAGS.corpus:
    train, test = tak.train.load_corpus(FLAGS.corpus, add_symmetries=FLAGS.symmetries)
  elif FLAGS.features:
    train, test = tak.train.load_features(FLAGS.features, FLAGS.size)
  else:
    print("You must specify --corpus or --features", out=sys.stderr)
    return 1

  model_def = tak.proto.ModelDef(
    size    = 5,

    layers  = FLAGS.layers,
    kernel  = FLAGS.kernel,
    filters = FLAGS.filters,
    hidden  = FLAGS.hidden,
  )
  test_batch = test.batch(int(1e5))
  iterator = tf.data.Iterator.from_structure(
    test_batch.output_types,
    test_batch.output_shapes)
  init_test = iterator.make_initializer(test_batch)
  init_train = iterator.make_initializer(train.shuffle(int(1e5)).batch(FLAGS.batch))
  next_batch = iterator.get_next()
  model = tak.model.PredictionModel(model_def, next_batch['position'])
  learning_rate = tf.placeholder(tf.float32)
  optimizer = getattr(tf.train, FLAGS.optimizer)(learning_rate)
  model.add_train_ops(next_batch['move'],
                      optimizer=optimizer, regularize=FLAGS.regularize)

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

  n_instances = tf.Variable(0)
  init_n = tf.assign(n_instances, 0)
  inc_n = tf.assign_add(n_instances, tf.shape(next_batch['position'])[0])

  for epoch in range(FLAGS.epochs):
    session.run(init_test)
    loss, prec1, prec5 = session.run(
      [model.loss, model.prec1, model.prec5],
      feed_dict={
        model.keep_prob: 1.0,
      })
    print("epoch={0} test loss={1:0.4f} acc={2:0.2f}%/{3:0.2f}% pos/s={4:.2f}".format(
      epoch, loss,
      100*prec1, 100*prec5,
      session.run(n_instances)/(t_end-t_start) if t_start else 0))
    if FLAGS.checkpoint:
      saver.save(session, FLAGS.checkpoint, global_step=epoch)

    t_start = time.time()
    session.run((init_train, init_n))
    while True:
      try:
        session.run(
          (model.train_step, inc_n),
          feed_dict={
            learning_rate: lr,
            model.keep_prob: FLAGS.dropout,
          })
      except tf.errors.OutOfRangeError:
        break
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
