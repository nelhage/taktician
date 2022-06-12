import os.path
import pickle
import tempfile

import pytest
import torch

from xformer import data

N_TEST = 16


@pytest.fixture(scope="module")
def encoded_dataset():
    with tempfile.TemporaryDirectory() as tmp:
        dataset_path = os.path.join(tmp, "dataset.pt")
        torch.save(
            {
                "ints": torch.arange(N_TEST),
                "squares": torch.arange(N_TEST) ** 2,
            },
            dataset_path,
        )
        yield dataset_path


def test_basic(encoded_dataset):
    for batch_size in (2, 4):
        ds = data.Dataset(encoded_dataset, batch_size=batch_size, seed=0x12345678)
        batches = list(ds)
        for b in batches:
            assert isinstance(b, data.Batch)
            assert set(b.data.keys()) == {"ints", "squares"}
            assert b.data["ints"].shape == (batch_size,)
            assert b.data["squares"].shape == (batch_size,)
            assert torch.equal(b.data["ints"] ** 2, b.data["squares"])
        all_ints = torch.cat([b.data["ints"] for b in batches])
        assert torch.equal(torch.sort(all_ints).values, torch.arange(N_TEST))


def test_odd_batch(encoded_dataset):
    ds = data.Dataset(encoded_dataset, batch_size=6, seed=0x12345678)
    batches = list(ds)
    assert [len(b.data["ints"]) for b in batches] == [6, 6, 4]


def test_reshuffle(encoded_dataset):
    ds = data.Dataset(encoded_dataset, batch_size=6, seed=0x12345678)
    b1 = list(ds)
    b2 = list(ds)

    i1 = torch.cat([b.data["ints"] for b in b1])
    i2 = torch.cat([b.data["ints"] for b in b2])
    assert not torch.equal(i1, i2)
    assert torch.equal(torch.sort(i1).values, torch.sort(i2).values)


def assert_same_data(ds1, ds2):
    for (l, r) in zip(iter(ds1), iter(ds2)):
        assert torch.equal(l.data["ints"], r.data["ints"])
        assert torch.equal(l.data["squares"], r.data["squares"])


def test_determinism(encoded_dataset):
    ds1 = data.Dataset(encoded_dataset, batch_size=2, seed=0x12345678)
    ds2 = data.Dataset(encoded_dataset, batch_size=2, seed=0x12345678)

    assert_same_data(ds1, ds2)


def test_fast_forward(encoded_dataset):
    ds1 = data.Dataset(encoded_dataset, batch_size=2, seed=0x12345678)
    ds2 = data.Dataset(encoded_dataset, batch_size=2, seed=0x12345678)

    for _ in range(4):
        list(ds1)
    ds2.fastforward_epochs(4)

    assert_same_data(ds1, ds2)


def test_serde(encoded_dataset):
    ds1 = data.Dataset(encoded_dataset, batch_size=2, seed=0x12345678)
    ds2 = data.Dataset(encoded_dataset, batch_size=2, seed=0x12345678)

    list(ds1)
    list(ds2)

    ds2 = pickle.loads(pickle.dumps(ds2))
    ds2.fastforward_epochs(1)

    assert_same_data(ds1, ds2)
