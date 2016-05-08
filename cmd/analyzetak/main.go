package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/cli"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

var (
	depth     = flag.Int("depth", 5, "minimax depth")
	all       = flag.Bool("all", false, "show all possible moves")
	tps       = flag.Bool("tps", false, "render position in tps")
	move      = flag.Int("move", 0, "PTN move number to analyze")
	timeLimit = flag.Duration("limit", time.Minute, "limit of how much time to use")
	black     = flag.Bool("black", false, "only analyze black's move")
	white     = flag.Bool("white", false, "only analyze white's move")
	seed      = flag.Int64("seed", 0, "specify a seed")
	debug     = flag.Int("debug", 1, "debug level")
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
	color := tak.NoColor
	switch {
	case *white && *black:
		log.Fatal("-white and -black are exclusive")
	case *white:
		color = tak.White
	case *black:
		color = tak.Black
	case *move != 0:
		color = tak.White
	}
	p, e := parsed.PositionAtMove(*move, color)
	if e != nil {
		log.Fatal("find move:", e)
	}

	analyze(p)
}

func analyze(p *tak.Position) {
	player := ai.NewMinimax(ai.MinimaxConfig{
		Size:  p.Size(),
		Depth: *depth,
		Seed:  *seed,
		Debug: *debug,
	})
	pv, val, _ := player.Analyze(p, *timeLimit)
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

	for _, m := range pv {
		p, _ = p.Move(&m)
	}

	fmt.Println("Resulting position:")
	cli.RenderBoard(os.Stdout, p)

	fmt.Println()
	fmt.Println()
}
