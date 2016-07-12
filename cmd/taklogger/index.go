package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/nelhage/taktician/logs"
	"github.com/nelhage/taktician/ptn"
)

func indexPTN(dir string, db string) error {
	ptns, e := readPTNs(dir)
	if e != nil {
		return e
	}

	os.Remove(db)
	repo, err := logs.Open(db)
	if err != nil {
		return fmt.Errorf("open: %v", err)
	}
	defer repo.Close()
	var gs []*logs.Game
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
		gs = append(gs, &logs.Game{
			Day:       day,
			ID:        id,
			Timestamp: t,
			Size:      size,
			Player1:   player1,
			Player2:   player2,
			Result:    result,
			Winner:    winner,
			Moves:     moves,
		})
	}
	err = repo.InsertGames(gs)
	if err != nil {
		return fmt.Errorf("insert: %v", err)
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
