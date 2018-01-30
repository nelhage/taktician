import tak.train
import tak.ptn
import tak.symmetry

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
    assert f[0,4,0] == 1.0
    assert f[4,0,1] == 1.0

    assert np.all(f[:,:,2:14] == 0)

    b1 = tak.ptn.parse_tps(
      '1,x4/x5/x5/x5/x4,2 2 2')
    f1 = tak.train.features(b1)
    assert np.sum(f1[:,:,0]) == 1
    assert np.sum(f1[:,:,1]) == 1
    assert f1[0,4,1] == 1.0
    assert f1[4,0,0] == 1.0

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

  def test_symmetry_features(self):
    pos = tak.ptn.parse_tps("2,x,21S,2,2,2/2,2C,2,1S,x2/x3,2,x2/1,11112,1121,1C,x2/x2,1S,12,1,1/x3,1,x,1 1 20")
    feat = tak.train.Featurizer(pos.size)

    manual = [
      feat.features(tak.symmetry.transform_position(sym, pos))
      for sym in tak.symmetry.SYMMETRIES
    ]
    computed = feat.features_symmetries(pos)
    for i in range(len(manual)):
      assert np.all(manual[i] == computed[i])
