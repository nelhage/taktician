package logs

const createGameTable = `
CREATE TABLE IF NOT EXISTS games (
  day string not null,
  id integer not null,
  time datetime,
  size int,
  player1 varchar,
  player2 varchar,
  result string,
  winner string,
  moves int
)`

const createPlayerTable = `
CREATE VIEW IF NOT EXISTS player_games (
  day, id, player, opponent, win
) AS
SELECT day, id, player2, player1,
 CASE winner WHEN 'white' THEN 'lose' WHEN 'black' THEN 'win' ELSE 'tie' END FROM games
UNION
SELECT day, id, player1, player2,
 CASE winner WHEN 'white' THEN 'win' WHEN 'black' THEN 'lose' ELSE 'tie' END FROM games
`

const insertStmt = `
INSERT INTO games (day, id, time, size, player1, player2, result, winner, moves)
VALUES (?,?,?,?,?,?,?,?,?)
`
