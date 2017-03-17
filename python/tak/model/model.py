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
          )
          self.layers.append(activations)

      outputs = self.size*self.size*model_def.filters
      if model_def.hidden > 0:
        self.W_h = tf.Variable(tf.zeros([outputs, model_def.hidden]), name="weights")
        self.b_h = tf.Variable(tf.zeros([model_def.hidden]), name="biases")

        x = tf.reshape(activations, [-1, outputs])
        activations = tf.nn.relu(tf.matmul(x, self.W_h) + self.b_h)
        outputs = model_def.hidden

    self.keep_prob = tf.placeholder_with_default(
      tf.ones(()), shape=(), name='keep_prob')

    self.output = tf.nn.dropout(activations, keep_prob=self.keep_prob)
    self.output_count = outputs

class PredictionModel(object):
  def __init__(self, model_def, perception=None):
    self.size = model_def.size

    if perception is None:
      fshape = tak.train.feature_shape(self.size)
      with tf.variable_scope('Input'):
        self.x = tf.placeholder(tf.float32, (None,) + fshape)
      perception = PerceptionModel(model_def, self.x)
    else:
      self.x = perception.x

    self.perception = perception

    self.move_count = tak.train.move_count(self.size)
    self.keep_prob = perception.keep_prob

    with tf.variable_scope('Output'):
      self.W = tf.Variable(tf.zeros([perception.output_count, self.move_count]), name="weights")
      self.b = tf.Variable(tf.zeros([self.move_count]), name="biases")

      x = tf.reshape(perception.output, [-1, perception.output_count])
      self.logits = tf.matmul(x, self.W) + self.b
    tf.add_to_collection('logits', self.logits)

  def add_train_ops(self, regularize=0, optimizer=tf.train.GradientDescentOptimizer.__name__):
    with tf.variable_scope('Input'):
      self.labels = tf.placeholder(tf.float32, (None, self.move_count))

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
