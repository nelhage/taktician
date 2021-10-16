package importptn

const createPTNTable = `
CREATE TABLE IF NOT EXISTS ptns (
  id integer primary key,
  ptn string
)
`

/*
CREATE TABLE games (
   id INTEGER PRIMARY KEY,
   date INT,
   size INT,
   player_white VARCHAR(20),
   player_black VARCHAR(20),
   notation TEXT,
   result VARCAR(10),
   timertime INT DEFAULT 0,
   timerinc INT DEFAULT 0,
   rating_white int default 1000,
   rating_black int default 1000,
   unrated int default 0,
   tournament int default 0,
   komi int default 0,
   pieces int default -1,
   capstones int default -1,
   rating_change_white int default 0,
   rating_change_black int default 0);
*/

type gameRow struct {
	Id   int `db:"id"`
	Date int `db:"date"`
	Size int `db:"size"`

	PlayerWhite string `db:"player_white"`
	PlayerBlack string `db:"player_black"`

	Notation string `db:"notation"`
	Result   string `db:"result"`

	TimerTime int `db:"timertime"`
	TimerInc  int `db:"timerinc"`
}

type ptnRow struct {
	Id  int    `db:"id"`
	PTN string `db:"ptn"`
}

const selectTODO = `
SELECT g.id, g.date, g.size, g.player_white, g.player_black, g.notation, g.result, g.timertime, g.timerinc
FROM games g LEFT OUTER JOIN ptns p
  ON (g.id = p.id)
WHERE p.id is NULL
  AND g.notation IS NOT NULL
  AND g.notation != ""
`

const insertPTN = `
INSERT INTO ptns (id, ptn)
VALUES (:id, :ptn)
`
