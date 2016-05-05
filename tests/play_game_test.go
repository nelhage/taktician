package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"

	"github.com/nelhage/taktician/cli"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

var games = flag.String("games", "", "Directory of .ptn files to self-check on")

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
		out = append(out, g)
	}
	return out, nil
}

func TestPlayPTNs(t *testing.T) {
	if *games == "" {
		t.SkipNow()
	}
	ptns, err := readPTNs(*games)
	if err != nil {
		t.Fatalf("read ptns: %v", err)
	}
	for _, p := range ptns {
		playPTN(t, p)
	}
}

func playPTN(t *testing.T, p *ptn.PTN) {
	id := p.FindTag("Id")
	if id == "" {
		return
	}
	t.Log("playing", id)
	size, _ := strconv.Atoi(p.FindTag("Size"))
	g := tak.New(tak.Config{Size: size})
	for _, op := range p.Ops {
		if m, ok := op.(*ptn.Move); ok {
			next, e := g.Move(&m.Move)
			if e != nil {
				fmt.Printf("illegal move: %s\n", ptn.FormatMove(&m.Move))
				fmt.Printf("move=%d\n", g.MoveNumber())
				cli.RenderBoard(os.Stdout, g)
				t.Fatal("illegal move")
			}
			g = next
		}
	}
	over, winner := g.GameOver()
	var d tak.WinDetails
	if over {
		d = g.WinDetails()
	}
	switch p.FindTag("Result") {
	case "R-0":
		if !over || winner != tak.White || d.Reason != tak.RoadWin {
			t.Error("road win for white:", d)
		}
	case "0-R":
		if !over || winner != tak.Black || d.Reason != tak.RoadWin {
			t.Error("road win for white:", d)
		}
	case "F-0":
		if !over || winner != tak.White || d.Reason != tak.FlatsWin {
			t.Error("flats win for white:", d)
		}
	case "0-F":
		if !over || winner != tak.Black || d.Reason != tak.FlatsWin {
			t.Error("flats win for black:", d)
		}
	case "1/2-1/2":
		/*
			if over && winner != tak.NoColor {
				t.Error("tie", over, d)
			}
		*/

		// playtak mishandles double-road wins as ties, so we
		// can't usefully check here.
	}
}
