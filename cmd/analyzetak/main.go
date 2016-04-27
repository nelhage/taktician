package main

import (
	"fmt"
	"log"
	"os"

	"nelhage.com/tak/ai"
	"nelhage.com/tak/cli"
	"nelhage.com/tak/ptn"
)

func main() {
	f, e := os.Open(os.Args[1])
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
	for _, op := range parsed.Ops {
		if m, ok := op.(*ptn.Move); ok {
			next, e := p.Move(m.Move)
			if e != nil {
				fmt.Printf("illegal move: %s\n", ptn.FormatMove(&m.Move))
				fmt.Printf("move=%d\n", p.MoveNumber())
				cli.RenderBoard(os.Stdout, p)
				log.Fatal("illegal move")
			}
			p = next
		}
	}
	player := ai.NewMinimax(5)
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
}
