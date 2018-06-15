import tak.ptn
import tak.train

import sys
import os.path

import numpy as np
import tensorflow as tf

def protoize(features):
  out = {}
  for k,v in features.items():
    if v.dtype == np.float32 or v.dtype == np.float64:
      feat = tf.train.Feature(float_list=tf.train.FloatList(value=v.flatten()))
    else:
      raise "unknown dtype"
    out[k] = feat
  return out

def main(args):
  if len(args) != 3:
    print("Usage: {} CORPUS FEATURES".format(args[0]), file=sys.stderr)
    return 1

  corpus, features = args[1:]

  train, test = tak.train.load_corpus(corpus)

  try:
    os.makedirs(features)
  except OSError:
    pass

  session = tf.InteractiveSession()
  write_dataset(session, train, os.path.join(features, "train.tfrecord"))
  write_dataset(session, test, os.path.join(features, "test.tfrecord"))

def write_dataset(session, dataset, path):
  next_row = dataset.make_one_shot_iterator().get_next()
  with tf.python_io.TFRecordWriter(path) as writer:
    try:
      while True:
        features = session.run(next_row)
        example = tf.train.Example(
          features = tf.train.Features(
            feature = protoize(features)
          )
        )
        writer.write(example.SerializeToString())
    except tf.errors.OutOfRangeError:
      pass


if __name__ == '__main__':
  tf.app.run(main=main, argv=sys.argv)
