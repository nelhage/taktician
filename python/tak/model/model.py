import tak.train

import tensorflow as tf

class PerceptionModel(object):
  def __init__(self, model_def, x):
    self.size = model_def.size
    self.x = x
    tf.add_to_collection('inputs', self.x)

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
            activation_fn=tf.nn.relu,
          )
          self.layers.append(activations)

      if model_def.hidden > 0:
        activations = tf.contrib.layers.fully_connected(
          tf.contrib.layers.flatten(activations),
          scope = 'Hidden',
          num_outputs = model_def.hidden,
          activation_fn = tf.nn.relu,
        )
    activations = tf.contrib.layers.flatten(activations)

    self.keep_prob = tf.placeholder_with_default(
      tf.ones(()), shape=(), name='keep_prob')

    self.output = tf.nn.dropout(activations, keep_prob=self.keep_prob)

class PredictionModel(object):
  def __init__(self, model_def, x):
    self.size = model_def.size

    fshape = tak.train.feature_shape(self.size)
    self.perception = PerceptionModel(model_def, x)

    self.move_count = tak.train.move_count(self.size)
    self.keep_prob = self.perception.keep_prob

    self.logits = tf.contrib.layers.fully_connected(
      tf.contrib.layers.flatten(self.perception.output),
      scope = 'Output',
      num_outputs = self.move_count,
      activation_fn = None,
    )
    tf.add_to_collection('logits', self.logits)

  def add_train_ops(self, labels, optimizer, regularize=0):
    self.labels = labels

    with tf.variable_scope('Train'):
      self.cross_entropy = tf.reduce_mean(
        tf.nn.softmax_cross_entropy_with_logits_v2(logits=self.logits, labels=self.labels))
      self.regularization_loss = tf.contrib.layers.apply_regularization(
        tf.contrib.layers.l2_regularizer(regularize),
      )

      self.loss = self.cross_entropy + self.regularization_loss
      self.global_step = tf.Variable(0, name='global_step', trainable=False)
      self.train_step = optimizer.minimize(self.loss, global_step=self.global_step)

      labels = tf.argmax(self.labels, 1)
      self.prec1 = tf.reduce_mean(tf.cast(
        tf.nn.in_top_k(self.logits, labels, 1), tf.float32))
      self.prec5 = tf.reduce_mean(tf.cast(
        tf.nn.in_top_k(self.logits, labels, 5), tf.float32))
