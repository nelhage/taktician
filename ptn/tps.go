package ptn

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/nelhage/taktician/tak"
)

func ParseTPS(tpn string) (*tak.Position, error) {
	var pieces [][]tak.Square
	words := strings.Split(tpn, " ")
	if len(words) != 3 {
		return nil, errors.New("bad TPN: wrong number of words")
	}
	turn, err := strconv.Atoi(words[1])
	if err != nil {
		return nil, fmt.Errorf("bad turn: %s", words[1])
	}
	if turn != 1 && turn != 2 {
		return nil, fmt.Errorf("bad turn: %s", words[1])
	}
	move, err := strconv.Atoi(words[2])
	if err != nil {
		return nil, fmt.Errorf("bad move: %s", words[2])
	}
	move = 2*(move-1) + (turn - 1)

	rows := strings.Split(words[0], "/")
	for _, r := range rows {
		row, err := parseRow(r)
		if err != nil {
			return nil, err
		}
		pieces = append([][]tak.Square{row}, pieces...)
	}
	if len(pieces) < 3 || len(pieces) > 8 {
		return nil, fmt.Errorf("bad size board: %d", len(pieces))
	}
	for i, r := range pieces {
		if len(r) != len(pieces) {
			return nil, fmt.Errorf("row %d bad length: %d", i, len(r))
		}
	}
	return tak.FromSquares(tak.Config{Size: len(pieces)}, pieces, move)
}

func FormatTPS(p *tak.Position) string {
	var rows []string
	for i := p.Size() - 1; i >= 0; i-- {
		rows = append(rows, tpsRow(p, i))
	}
	var toMove string
	if p.ToMove() == tak.White {
		toMove = "1"
	} else {
		toMove = "2"
	}
	return fmt.Sprintf("%s %s %d", strings.Join(rows, "/"), toMove, p.MoveNumber()/2+1)
}

func tpsRow(p *tak.Position, y int) string {
	var bits []string
	for x := 0; x < p.Size(); {
		var i int
		for i = 0; x+i < p.Size() && len(p.At(x+i, y)) == 0; i++ {
		}
		switch i {
		case 0:
			bits = append(bits, tpsSquare(p.At(x, y)))
			x++
		case 1:
			bits = append(bits, "x")
		default:
			bits = append(bits, fmt.Sprintf("x%d", i))
		}
		x += i
	}
	return strings.Join(bits, ",")
}

func tpsSquare(sq tak.Square) string {
	var out []byte
	for i := len(sq) - 1; i >= 0; i-- {
		if sq[i].Color() == tak.White {
			out = append(out, '1')
		} else {
			out = append(out, '2')
		}
	}
	if sq[0].Kind() == tak.Standing {
		out = append(out, 'S')
	} else if sq[0].Kind() == tak.Capstone {
		out = append(out, 'C')
	}
	return string(out)
}

func parseRow(row string) ([]tak.Square, error) {
	var out []tak.Square
	bits := strings.Split(row, ",")
	for _, bit := range bits {
		if bit[0] == 'x' {
			count := 1
			if len(bit) > 1 {
				count = int(bit[1] - '0')
			}
			for i := 0; i < count; i++ {
				out = append(out, nil)
			}
			continue
		}
		stack := make(tak.Square, len(bit))
		for i, b := range bit {
			switch b {
			case '1':
				stack[len(stack)-i-1] = tak.MakePiece(tak.White, tak.Flat)
			case '2':
				stack[len(stack)-i-1] = tak.MakePiece(tak.Black, tak.Flat)
			case 'C', 'S':
				if i != len(bit)-1 {
					return nil, fmt.Errorf("stone type not at end of stack: %s", bit)
				}
				stack = stack[1:]
				color := stack[0].Color()
				if b == 'S' {
					stack[0] = tak.MakePiece(color, tak.Standing)
				} else {
					stack[0] = tak.MakePiece(color, tak.Capstone)
				}
			default:
				return nil, fmt.Errorf("malformed stack: %s", bit)
			}
		}
		out = append(out, stack)
	}
	return out, nil
}
