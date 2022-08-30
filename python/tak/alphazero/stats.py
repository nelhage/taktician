from attrs import define, field
from typing import Any


@define(slots=False)
class Elapsed:
    step: int = 0
    positions: int = 0
    epoch: int = 0
