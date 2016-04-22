package main

import (
	"bufio"
	"fmt"
	"io"
	"text/tabwriter"

	"nelhage.com/tak/game"
	"nelhage.com/tak/ptn"
)

type state struct {
	p   *game.Position
	out io.Writer
	in  *bufio.Reader
}

func playTak(st *state) {
	for {
		drawGame(st)
		if ok, c := st.p.GameOver(); ok {
			fmt.Fprintln(st.out, "Game over! Winner:", c)
			return
		}
		m := readMove(st)
		p, e := st.p.Move(*m)
		if e != nil {
			fmt.Fprintln(st.out, "illegal move:", e)
		} else {
			st.p = p
		}
	}

}

func drawGame(st *state) {
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

func readMove(st *state) *game.Move {
	for {
		fmt.Printf("%s> ", st.p.ToMove())
		line, err := st.in.ReadString('\n')
		if err != nil {
			panic(err)
		}
		m, err := ptn.ParseMove(line)
		if err != nil {
			fmt.Fprintln(st.out, "parse error: ", err)
			continue
		}
		return m
	}
}
