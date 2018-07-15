import tak.train

import tensorflow as tf

class PerceptionModel(object):
  def __init__(self, model_def, x):
    self.size = model_def.size

    with tf.variable_scope('Hidden'):
      activations = x
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
