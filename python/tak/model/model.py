import tak.train

import tensorflow as tf

class Model(object):
  def __init__(self, model_def):
    self.size = model_def.size

    fshape = tak.train.feature_shape(self.size)
    _, _, fplanes = fshape
    fcount = fplanes * self.size * self.size
    mcount = tak.train.move_count(self.size)

    with tf.variable_scope('Input'):
      self.x = tf.placeholder(tf.float32, (None,) + fshape)
      self.labels = tf.placeholder(tf.float32, (None, mcount))

    with tf.variable_scope('Hidden'):
      activations = self.x
      self.layers = []
      for i in range(model_def.layers):
        with tf.variable_scope('Layer{0}'.format(i)):
          activations = tf.contrib.layers.convolution2d(
            activations,
            num_outputs=model_def.filters,
            padding='SAME',
            kernel_size=model_def.kernel,
            trainable=True,
            variables_collections={'weights': [tf.GraphKeys.WEIGHTS]},
          )
          self.layers.append(activations)

      icount = self.size*self.size*model_def.filters
      if model_def.hidden > 0:
        self.W_h = tf.Variable(tf.zeros([icount, model_def.hidden]), name="weights")
        self.b_h = tf.Variable(tf.zeros([model_def.hidden]), name="biases")

        x = tf.reshape(activations, [-1, icount])
        activations = tf.nn.relu(tf.matmul(x, self.W_h) + self.b_h)
        icount = model_def.hidden

    with tf.variable_scope('Output'):
      self.keep_prob = tf.placeholder_with_default(
        tf.ones(()), shape=(), name='keep_prob')
      self.W = tf.Variable(tf.zeros([icount, mcount]), name="weights")
      self.b = tf.Variable(tf.zeros([mcount]), name="biases")

      x = tf.reshape(tf.nn.dropout(activations, keep_prob=self.keep_prob),
                     [-1, icount])
      self.logits = tf.matmul(x, self.W) + self.b
    tf.add_to_collection('inputs', self.x)
    tf.add_to_collection('logits', self.logits)

  def add_train_ops(self, regularize=0, optimizer=tf.train.GradientDescentOptimizer.__name__):
    with tf.variable_scope('Train'):
      self.cross_entropy = tf.reduce_mean(
        tf.nn.softmax_cross_entropy_with_logits(logits=self.logits, labels=self.labels))
      self.regularization_loss = tf.contrib.layers.apply_regularization(
        tf.contrib.layers.l2_regularizer(regularize),
      )

      self.loss = self.cross_entropy + self.regularization_loss
      self.global_step = tf.Variable(0, name='global_step', trainable=False)
      self.learning_rate = tf.placeholder(tf.float32)
      self.train_step = (getattr(tf.train, optimizer)(self.learning_rate).
                         minimize(self.loss, global_step=self.global_step))

      labels = tf.argmax(self.labels, 1)
      self.prec1 = tf.reduce_mean(tf.cast(
        tf.nn.in_top_k(self.logits, labels, 1), tf.float32))
      self.prec5 = tf.reduce_mean(tf.cast(
        tf.nn.in_top_k(self.logits, labels, 5), tf.float32))
