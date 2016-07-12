package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/nelhage/taktician/ptn"
)

const createGameTable = `
CREATE TABLE games (
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
CREATE VIEW player_games (
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

func indexPTN(dir string, db string) error {
	ptns, e := readPTNs(dir)
	if e != nil {
		return e
	}

	os.Remove(db)
	sql, err := sql.Open("sqlite3", db)
	if err != nil {
		return err
	}
	defer sql.Close()
	_, err = sql.Exec(createGameTable)
	if err != nil {
		return fmt.Errorf("create game table: %v", err)
	}
	_, err = sql.Exec(createPlayerTable)
	if err != nil {
		return fmt.Errorf("create player_game table: %v", err)
	}
	tx, err := sql.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(insertStmt)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	for _, g := range ptns {
		day := g.FindTag("Date")
		id, e := strconv.Atoi(g.FindTag("Id"))
		if day == "" || e != nil {
			continue
		}
		size, _ := strconv.Atoi(g.FindTag("Size"))
		t, _ := time.Parse(g.FindTag("Time"), time.RFC3339)
		player1 := g.FindTag("Player1")
		player2 := g.FindTag("Player2")
		result := g.FindTag("Result")
		winner := (&ptn.Result{Result: result}).Winner().String()
		moves := countMoves(g)
		_, e = stmt.Exec(
			day, id, t, size, player1, player2, result, winner, moves,
		)
		if e != nil {
			return e
		}
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("commit: %v", err)
	}

	return nil
}

func countMoves(g *ptn.PTN) int {
	i := 0
	for _, o := range g.Ops {
		if _, ok := o.(*ptn.Move); ok {
			i++
		}
	}
	return i
}

func readPTNs(d string) ([]*ptn.PTN, error) {
	var out []*ptn.PTN
	e := filepath.Walk(d, func(path string, info os.FileInfo, err error) error {
		if !strings.HasSuffix(path, ".ptn") {
			return nil
		}
		f, e := os.Open(path)
		if e != nil {
			log.Printf("open(%s): %v", path, e)
			return nil
		}
		defer f.Close()
		g, e := ptn.ParsePTN(f)
		if e != nil {
			log.Printf("parse(%s): %v", path, e)
			return nil
		}
		out = append(out, g)
		return nil
	})
	return out, e
}
