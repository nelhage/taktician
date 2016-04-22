package main

import (
	"bufio"
	"fmt"
	"io"
	"text/tabwriter"

	"nelhage.com/tak/game"
	"nelhage.com/tak/ptn"
)

type Player interface {
	GetMove(p *game.Position) *game.Move
}

type cliPlayer struct {
	out io.Writer
	in  *bufio.Reader
}

type state struct {
	p     *game.Position
	out   io.Writer
	white Player
	black Player
}

func playTak(st *state) {
	var moves []*game.Move
	for {
		drawGame(st)
		if ok, c := st.p.GameOver(); ok {
			fmt.Fprintln(st.out, "Game over! Winner:", c)
			return
		}
		var m *game.Move
		if st.p.ToMove() == game.White {
			m = st.white.GetMove(st.p)
		} else {
			m = st.black.GetMove(st.p)
		}
		p, e := st.p.Move(*m)
		if e != nil {
			fmt.Fprintln(st.out, "illegal move:", e)
		} else {
			st.p = p
			moves = append(moves, m)
		}
		if len(moves)%2 == 0 {
			fmt.Fprintf(st.out,
				"%d. %s  %s\n",
				len(moves)/2,
				ptn.FormatMove(moves[len(moves)-2]),
				ptn.FormatMove(moves[len(moves)-1]))
		}
	}

}

func drawGame(st *state) {
	fmt.Fprintln(st.out)
	w := tabwriter.NewWriter(st.out, 4, 8, 1, '\t', 0)
	for y := st.p.Size() - 1; y >= 0; y-- {
		fmt.Fprintf(w, "%c.\t", '1'+y)
		for x := 0; x < st.p.Size(); x++ {
			fmt.Fprintf(w, "%v\t", st.p.At(x, y))
		}
		fmt.Fprintf(w, "\n")
	}
	fmt.Fprintf(w, "\t")
	for x := 0; x < st.p.Size(); x++ {
		fmt.Fprintf(w, "%c.\t", 'a'+x)
	}
	fmt.Fprintf(w, "\n")
	w.Flush()
}

func (c *cliPlayer) GetMove(p *game.Position) *game.Move {
	for {
		fmt.Fprintf(c.out, "%s> ", p.ToMove())
		line, err := c.in.ReadString('\n')
		if err != nil {
			panic(err)
		}
		m, err := ptn.ParseMove(line)
		if err != nil {
			fmt.Fprintln(c.out, "parse error: ", err)
			continue
		}
		return m
	}
}
