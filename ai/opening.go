package ai

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/nelhage/taktician/canonicalize"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

type OpeningBook struct {
	size int
	book map[uint64]*openingPosition
}

type openingPosition struct {
	p     *tak.Position
	moves []child
}

type child struct {
	move   tak.Move
	weight int
}

func BuildOpeningBook(size int, lines []string) (*OpeningBook, error) {
	ob := &OpeningBook{
		size: size,
		book: make(map[uint64]*openingPosition),
	}
	for lno, line := range lines {
		p := tak.New(tak.Config{Size: size})
		bits := strings.Split(line, " ")

		for _, b := range bits {
			m, e := ptn.ParseMove(b)
			if e != nil {
				return nil, fmt.Errorf("line %d: move `%s`: %v",
					lno, b, e)
			}

			rs, e := canonicalize.Symmetries(p)
			if e != nil {
				return nil, fmt.Errorf("compute symmetries: %v", e)
			}
			for _, sym := range rs {
				pos, ok := ob.book[sym.P.Hash()]
				if !ok {
					pos = &openingPosition{
						p: sym.P,
					}
					ob.book[sym.P.Hash()] = pos
				}
				sm := canonicalize.TransformMove(sym.S, m)
				var ch *child
				for i := range pos.moves {
					if pos.moves[i].move.Equal(sm) {
						ch = &pos.moves[i]
						break
					}
				}
				if ch == nil {
					pos.moves = append(pos.moves,
						child{
							move:   sm,
							weight: 0,
						})
					ch = &pos.moves[len(pos.moves)-1]
				}
				ch.weight++
			}

			p, e = p.Move(m)
			if e != nil {
				return nil, fmt.Errorf("line %d: move `%s`: %v",
					lno, b, e)
			}
		}
	}

	return ob, nil
}

func (ob *OpeningBook) GetMove(p *tak.Position, r *rand.Rand) (tak.Move, bool) {
	pos, ok := ob.book[p.Hash()]
	if !ok {
		return tak.Move{}, false
	}
	sum := 0
	var out tak.Move
	for _, ch := range pos.moves {
		sum += ch.weight
		if r.Int31n(int32(sum)) < int32(ch.weight) {
			out = ch.move
		}
	}
	return out, true
}
