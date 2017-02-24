import tak.train
import tak.ptn

import numpy as np

class TestFeatures(object):
  def extra_planes(self, feat):
    return feat[:,:,14:]

  def is_onehot(self, m, axis=2):
    return np.all(np.sum(m, axis) == 1)

  def test_zero_features(self):
    b = tak.Position.from_config(tak.Config(size=5))
    f = tak.train.features(b)
    assert f.shape == tak.train.feature_shape(5)
    assert np.all(f[:,:,:14] == 0)

    assert np.all(f[:,:,16] == 1)


  def test_basic_features(self):
    b = tak.ptn.parse_tps(
      '1,x4/x5/x5/x5/x4,2 1 2')
    f = tak.train.features(b)
    assert np.sum(f[:,:,0]) == 1
    assert np.sum(f[:,:,1]) == 1
    assert f[0,0,0] == 1.0
    assert f[4,4,1] == 1.0

    assert np.all(f[:,:,2:14] == 0)

    b1 = tak.ptn.parse_tps(
      '1,x4/x5/x5/x5/x4,2 2 2')
    f1 = tak.train.features(b1)
    assert np.sum(f1[:,:,0]) == 1
    assert np.sum(f1[:,:,1]) == 1
    assert f1[0,0,1] == 1.0
    assert f1[4,4,0] == 1.0

  def test_flats(self):
    f = tak.train.features(
      tak.ptn.parse_tps(
        '1,x4/x5/x5/x5/x4,2 1 2'))
    ext = self.extra_planes(f)
    assert self.is_onehot(
      ext[:,:, tak.train.FeaturePlane.FLATS:tak.train.FeaturePlane.FLATS_MAX],
    )
    assert np.all(ext[:,:, tak.train.FeaturePlane.FLATS + 3] == 1)

    f = tak.train.features(
      tak.ptn.parse_tps(
        '1,1,x3/x5/x5/x5/x4,2 1 2'))
    ext = self.extra_planes(f)
    assert self.is_onehot(
      ext[:,:, tak.train.FeaturePlane.FLATS:tak.train.FeaturePlane.FLATS_MAX],
    )
    assert np.all(ext[:,:, tak.train.FeaturePlane.FLATS + 4] == 1)

    f = tak.train.features(
      tak.ptn.parse_tps(
        '1,1,1,1,1/1,1,1,1,1/x5/x5/x4,2 1 2'))
    ext = self.extra_planes(f)
    assert self.is_onehot(
      ext[:,:, tak.train.FeaturePlane.FLATS:tak.train.FeaturePlane.FLATS_MAX],
    )
    assert np.all(ext[:,:, tak.train.FeaturePlane.FLATS_MAX-1] == 1)

    f = tak.train.features(
      tak.ptn.parse_tps(
        '1,1,1,1,1/1,1,1,1,1/x5/x5/x4,2 2 2'))
    ext = self.extra_planes(f)
    assert self.is_onehot(
      ext[:,:, tak.train.FeaturePlane.FLATS:tak.train.FeaturePlane.FLATS_MAX],
    )
    assert np.all(ext[:,:, tak.train.FeaturePlane.FLATS] == 1)
