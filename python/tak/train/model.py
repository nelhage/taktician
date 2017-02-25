import tensorflow as tf
import numpy as np

from .features import *

class TakModel(object):
  def __init__(self, graph, session, eval_symmetries):
    self.graph = graph
    self.session = session
    self.eval_symmetries = eval_symmetries

    self.input, = self.graph.get_collection('inputs')
    assert len(self.input.shape) == 4
    self.size = int(self.input.shape[1])
    assert self.input.shape[1:] == feature_shape(self.size)
    self.logits, = self.graph.get_collection('logits')
    self.softmax = tf.reduce_mean(tf.nn.softmax(self.logits), axis=0)

    self.buf = np.ndarray((1,) + feature_shape(self.size))

  def __call__(self, position):
    features(position, out=self.buf[0])

    return self.session.run(self.softmax, feed_dict={
      self.input: self.buf,
    })

def load_model(path, eval_symmetries=True):
  graph = tf.Graph()
  with graph.as_default():
    session = tf.Session()
    saver = tf.train.import_meta_graph(path + '.meta')
    saver.restore(session, path)
    return TakModel(graph, session, eval_symmetries)

__all__ = ['TakModel', 'load_model']
