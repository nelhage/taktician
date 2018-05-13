package taktest

import (
	"errors"
	"fmt"
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

func Board(tpl string, who tak.Color) (*tak.Position, error) {
	lines := strings.Split(strings.Trim(tpl, " \n"), "\n")
	var pieces [][]tak.Square
	for _, l := range lines {
		bits := strings.Split(l, " ")
		var row []tak.Square
		for _, p := range bits {
			switch p {
			case "W":
				row = append(row, tak.Square{tak.MakePiece(tak.White, tak.Flat)})
			case "B":
				row = append(row, tak.Square{tak.MakePiece(tak.Black, tak.Flat)})
			case "WC":
				row = append(row, tak.Square{tak.MakePiece(tak.White, tak.Capstone)})
			case "BC":
				row = append(row, tak.Square{tak.MakePiece(tak.Black, tak.Capstone)})
			case "WS":
				row = append(row, tak.Square{tak.MakePiece(tak.White, tak.Standing)})
			case "BS":
				row = append(row, tak.Square{tak.MakePiece(tak.Black, tak.Standing)})
			case ".":
				row = append(row, tak.Square{})
			case "":
			default:
				return nil, fmt.Errorf("bad piece: %v", p)
			}
		}
		if len(row) != len(lines) {
			return nil, errors.New("size mismatch")
		}
		pieces = append(pieces, row)
	}
	ply := 2
	if who == tak.Black {
		ply = 3
	}
	return tak.FromSquares(tak.Config{Size: len(pieces)}, pieces, ply)
}
