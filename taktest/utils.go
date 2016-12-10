package taktest

import (
	"strings"

	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

func Move(s string) tak.Move {
	m, e := ptn.ParseMove(s)
	if e != nil {
		panic(e)
	}
	return m
}

func Moves(s string) []tak.Move {
	if s == "" {
		return nil
	}
	bits := strings.Split(s, " ")
	var ms []tak.Move
	for _, b := range bits {
		m, e := ptn.ParseMove(b)
		if e != nil {
			panic(e)
		}
		ms = append(ms, m)
	}
	return ms
}

func FormatMoves(ms []tak.Move) string {
	var bits []string
	for _, o := range ms {
		bits = append(bits, ptn.FormatMove(o))
	}
	return strings.Join(bits, " ")
}

func Position(size int, ms string) *tak.Position {
	p := tak.New(tak.Config{Size: size})
	moves := Moves(ms)
	var e error
	for _, m := range moves {
		p, e = p.Move(m)
		if e != nil {
			panic(e)
		}
	}
	return p
}
