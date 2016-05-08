package tests

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/nelhage/taktician/cli"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

var games = flag.String("games", "", "Directory of .ptn files to self-check on")

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
