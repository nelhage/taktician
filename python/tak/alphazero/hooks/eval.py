from attrs import field, define
import subprocess
import os.path
import time
import tempfile
import shlex
import json
import math

from typing import Optional

from ..config import Config
from ..trainer import Hook, TrainState

SCRIPTS_DIR = os.path.realpath(
    os.path.join(os.path.dirname(__file__), "../../../scripts/")
)


@define(slots=False)
class EvalHook(Hook):
    opponent: str
    tei_opts: list[str] = ["--simulation-limit=1", "--argmax", "--time-limit=0"]

    name: str = "eval"

    model: Optional[tuple[str, int]] = None
    openings: Optional[str] = None

    frequency: int = 100

    def before_run(self, state, config: Config):
        self.config = config

    def after_step(self, state: TrainState):
        if state.elapsed.step > 1 and state.elapsed.step % self.frequency != 0:
            return

        print(f"Running eval name={self.name}...")

        model_proc = None
        if self.model is not None:
            (model, port) = self.model
            model_proc = subprocess.Popen(
                [
                    os.path.join(SCRIPTS_DIR, "analysis_server"),
                    "--port",
                    str(port),
                    "--device",
                    self.config.device,
                    model,
                ],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
            )
            time.sleep(1)
            if model_proc.poll():
                print(f"WARN: model {model} failed to start on :{port}!")
                return

        with tempfile.TemporaryDirectory() as tmp:
            summary_file = os.path.join(tmp, "summary.json")
            p1_cmd = [
                os.path.join(SCRIPTS_DIR, "tei"),
                "--host=localhost",
                f"--port={self.config.server_port}",
            ] + self.tei_opts
            selfplay_cmd = [
                "taktician",
                "selfplay",
                "-size",
                str(self.config.size),
                "-games=1",
                f"-summary={summary_file}",
            ]
            if self.openings is not None:
                selfplay_cmd += ["-openings", self.openings]
            selfplay_cmd += ["-p1", shlex.join(p1_cmd), "-p2", self.opponent]

            try:
                subprocess.check_call(selfplay_cmd)
            except subprocess.CalledProcessError:
                print("WARN: Unable to run evals!")
                return

            with open(summary_file, "r") as fh:
                stats = json.load(fh)

        # stats
        p1 = stats["Stats"]["Players"][0]
        p2 = stats["Stats"]["Players"][1]

        ties = stats["Stats"]["Ties"]

        games = (
            stats["Stats"]["White"]
            + stats["Stats"]["Black"]
            + stats["Stats"]["Ties"]
            + stats["Stats"]["Cutoff"]
        )

        score = (p1["Wins"] + ties / 2) / games
        if score == 0:
            elo = -float("inf")
        elif score == 1:
            elo = float("inf")
        else:
            elo = -400 * math.log10(1 / score - 1)

        state.step_stats.update(
            {
                f"{self.name}.win_rate": p1["Wins"] / games,
                f"{self.name}.elo": elo,
            }
        )
