import tensorflow as tf
import numpy as np

import tak.symmetry

from .features import *
from .moves import *

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
    self.softmax = tf.nn.softmax(self.logits)

    if eval_symmetries:
      self.buf = np.ndarray((8,) + feature_shape(self.size))
      self._compute_move_perms()
    else:
      self.buf = np.ndarray((1,) + feature_shape(self.size))

  def _compute_move_perms(self):
    self.move_permutations = np.ndarray((8, move_count(self.size)), dtype=np.intp)
    for si, sym in enumerate(tak.symmetry.SYMMETRIES):
      for mi in range(move_count(self.size)):
        tm = tak.symmetry.transform_move(sym, id2move(mi, self.size), self.size)
        tmid = move2id(tm, self.size)
        self.move_permutations[si, mi] = tmid

  def __call__(self, position):
    if self.eval_symmetries:
      for i,s in enumerate(tak.symmetry.SYMMETRIES):
        features(tak.symmetry.transform_position(s, position),
                 out=self.buf[i])
      probs = self.session.run(self.softmax, feed_dict={
        self.input: self.buf,
      })
      for (i,perm) in enumerate(self.move_permutations):
        probs[i] = probs[i][perm]
      return np.mean(probs, axis=0)
    else:
      features(position, out=self.buf[0])
      return self.session.run(self.softmax, feed_dict={
        self.input: self.buf,
      })[0]

def load_model(path, eval_symmetries=True):
  graph = tf.Graph()
  with graph.as_default():
    session = tf.Session()
    saver = tf.train.import_meta_graph(path + '.meta')
    saver.restore(session, path)
    return TakModel(graph, session, eval_symmetries)

__all__ = ['TakModel', 'load_model']
