package main

import (
	"database/sql"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"nelhage.com/tak/ptn"
)

const createTable = `
CREATE TABLE games (
  id integer not null primary key,
  size int,
  player1 varchar,
  player2 varchar,
  result string,
  winner string,
  moves int
)
`

const insertStmt = `
INSERT INTO games (id, size, player1, player2, result, winner, moves)
VALUES (?,?,?,?,?,?,?)
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
	_, err = sql.Exec(createTable)
	if err != nil {
		return err
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
		id, _ := strconv.Atoi(g.FindTag("Id"))
		size, _ := strconv.Atoi(g.FindTag("Size"))
		player1 := g.FindTag("Player1")
		player2 := g.FindTag("Player2")
		result := g.FindTag("Result")
		winner := (&ptn.Result{Result: result}).Winner().String()
		moves := countMoves(g)
		_, e := stmt.Exec(
			id, size, player1, player2, result, winner, moves,
		)
		if e != nil {
			return e
		}
	}
	return tx.Commit()
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
	ents, e := ioutil.ReadDir(d)
	if e != nil {
		return nil, e
	}
	var out []*ptn.PTN
	for _, de := range ents {
		if !strings.HasSuffix(de.Name(), ".ptn") {
			continue
		}
		id := strings.SplitN(de.Name(), ".", 2)[0]
		_, e := strconv.ParseInt(id, 10, 64)
		if e != nil {
			continue
		}
		f, e := os.Open(path.Join(d, de.Name()))
		if e != nil {
			log.Printf("open(%s): %v", de.Name(), e)
			continue
		}
		g, e := ptn.ParsePTN(f)
		if e != nil {
			log.Printf("parse(%s): %v", de.Name(), e)
			f.Close()
			continue
		}
		f.Close()
		if g.FindTag("Id") == "" {
			g.Tags = append(g.Tags, ptn.Tag{Name: "Id", Value: id})
		}
		ioutil.WriteFile(path.Join(d, de.Name()), []byte(g.Render()), 0644)
		out = append(out, g)
	}
	return out, nil
}
