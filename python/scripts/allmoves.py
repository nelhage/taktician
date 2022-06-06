import tak.ptn

import sqlite3
import os.path
import collections
import traceback

SIZE = 5
GAMES_DIR = os.path.join(os.path.dirname(__file__), "../../games")
DB = sqlite3.connect(os.path.join(GAMES_DIR, "games.db"))

cur = DB.cursor()
cur.execute("select day, id from games where size = ?", (SIZE,))

corpus = collections.Counter()

while True:
    row = cur.fetchone()
    if not row:
        break
    day, id = None, None
    try:
        day, id = row
        text = open(os.path.join(GAMES_DIR, day, str(id) + ".ptn")).read()
        ptn = tak.ptn.PTN.parse(text)
        for m in ptn.moves:
            corpus[m] += 1
    except Exception as e:
        print("{0}/{1}: {2}".format(day, id, e))
        traceback.print_exc()
        continue

all_moves = set(tak.enumerate_moves(SIZE))
seen_moves = set(corpus.keys())

total = sum(corpus.values())
print("observed {0} unique moves".format(len(corpus)))
print("failed to generate: ", [tak.ptn.format_move(m) for m in seen_moves - all_moves])
print("did not observe: ", [tak.ptn.format_move(m) for m in all_moves - seen_moves])
for k, v in sorted(corpus.items(), key=lambda p: -p[1])[:50]:
    print("{0:6.2f}% ({2:6d}) {1}".format(100 * v / total, tak.ptn.format_move(k), v))
