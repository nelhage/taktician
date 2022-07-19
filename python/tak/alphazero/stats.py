from attrs import define, field


@define(slots=False)
class Elapsed:
    step: int = 0
    positions: int = 0
    epoch: int = 0
