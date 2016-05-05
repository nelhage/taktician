package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/cli"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

var (
	depth = flag.Int("depth", 5, "minimax depth")
	all   = flag.Bool("all", false, "show all possible moves")
	tps   = flag.Bool("tps", false, "render position in tps")
	move  = flag.Int("move", 0, "PTN move number to analyze")
)

func main() {
	flag.Parse()

	f, e := os.Open(flag.Arg(0))
	if e != nil {
		log.Fatal("open:", e)
	}
	parsed, e := ptn.ParsePTN(f)
	if e != nil {
		log.Fatal("parse:", e)
	}
	p, e := parsed.InitialPosition()
	if e != nil {
		log.Fatal("analyze:", e)
	}
	found := false
	for _, op := range parsed.Ops {
		if n, ok := op.(*ptn.MoveNumber); ok && n.Number == *move {
			found = true
			analyze(p)
		}
		if m, ok := op.(*ptn.Move); ok {
			next, e := p.Move(&m.Move)
			if e != nil {
				fmt.Printf("illegal move: %s\n", ptn.FormatMove(&m.Move))
				fmt.Printf("move=%d\n", p.MoveNumber())
				cli.RenderBoard(os.Stdout, p)
				log.Fatal("illegal move")
			}
			p = next
			if found {
				break
			}
		}
	}
	analyze(p)
}

func analyze(p *tak.Position) {
	player := ai.NewMinimax(p.Size(), *depth)
	player.Debug = true
	pv, val := player.Analyze(p)
	cli.RenderBoard(os.Stdout, p)
	fmt.Printf("AI analysis:\n")
	fmt.Printf(" pv=")
	for _, m := range pv {
		fmt.Printf("%s ", ptn.FormatMove(&m))
	}
	fmt.Printf("\n")
	fmt.Printf(" value=%d\n", val)
	if *tps {
		fmt.Printf("[TPS \"%s\"]\n", ptn.FormatTPS(p))
	}
	if *all {
		fmt.Printf(" all moves:")
		for _, m := range p.AllMoves() {
			fmt.Printf(" %s", ptn.FormatMove(&m))
		}
		fmt.Printf("\n")
	}
	fmt.Println()
	fmt.Println()
}
