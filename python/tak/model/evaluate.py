import tensorflow as tf
import numpy as np

import tak.train

def multinomial(probs):
  r = np.random.random()
  s = 0
  for i,p in enumerate(probs):
    s += p
    if r < s:
      return i

class Evaluator(object):
  def __init__(self, graph, session, eval_symmetries):
    self.graph = graph
    self.session = session
    self.eval_symmetries = eval_symmetries

    self.input, = self.graph.get_collection('inputs')
    assert len(self.input.shape) == 4
    self.size = int(self.input.shape[1])
    self.logits, = self.graph.get_collection('logits')
    self.softmax = tf.nn.softmax(self.logits)

    self.features = tak.train.Featurizer(self.size)
    assert self.input.shape[1:] == self.features.feature_shape()

    if eval_symmetries:
      self.buf = np.ndarray((8,) + self.features.feature_shape())
    else:
      self.buf = np.ndarray((1,) + self.features.feature_shape())

  def evaluate(self, position):
    if self.eval_symmetries:
      self.features.features_symmetries(position, self.buf)
      probs = self.session.run(self.softmax, feed_dict={
        self.input: self.buf,
      })
      self.features.unpermute_moves(probs)
      p = np.sum(probs, axis=0)
      return p / np.sum(p)
    else:
      self.features.features(position, out=self.buf[0])
      return self.session.run(self.softmax, feed_dict={
        self.input: self.buf,
      })[0]

  def get_move(self, position):
    probs = self.evaluate(position)
    while True:
      i = multinomial(probs)
      m = self.features.id2move(i)
      try:
        return m, position.move(m)
      except tak.IllegalMove:
        p = probs[i]
        probs[i] = 0
        probs /= (1-p)

def load_model(path, eval_symmetries=True):
  graph = tf.Graph()
  with graph.as_default():
    session = tf.Session()
    saver = tf.train.import_meta_graph(path + '.meta')
    saver.restore(session, path)
    return Evaluator(graph, session, eval_symmetries)

__all__ = ['Evaluator', 'load_model']
