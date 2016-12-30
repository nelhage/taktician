package symmetry

import (
	"fmt"

	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

type Symmetry func(int8, int8) (int8, int8)

func compose(ss ...Symmetry) Symmetry {
	return func(x, y int8) (int8, int8) {
		for i := range ss {
			s := ss[len(ss)-i-1]
			x, y = s(x, y)
		}
		return x, y
	}
}

func symmetries(size int) []Symmetry {
	flip := func(i int8) int8 {
		return int8(size) - 1 - i
	}

	identity := func(x, y int8) (int8, int8) {
		return x, y
	}

	flipX := func(x, y int8) (int8, int8) {
		return flip(x), y
	}

	flipY := func(x, y int8) (int8, int8) {
		return x, flip(y)
	}
	flipDiag1 := func(x, y int8) (int8, int8) {
		return y, x
	}
	flipDiag2 := func(x, y int8) (int8, int8) {
		return flip(y), flip(x)
	}

	rotate2 := func(x, y int8) (int8, int8) {
		return flip(x), flip(y)
	}
	rotCW := func(x, y int8) (int8, int8) {
		return y, flip(x)
	}
	rotCCW := func(x, y int8) (int8, int8) {
		return flip(y), x
	}

	return []Symmetry{
		identity,
		flipX,
		flipY,
		flipDiag1,
		flipDiag2,
		rotate2,
		rotCW,
		rotCCW,
	}
}

func TransformMove(s Symmetry, m tak.Move) tak.Move {
	var out tak.Move
	out.X, out.Y = s(m.X, m.Y)
	if !m.IsSlide() {
		out.Type = m.Type
		return out
	}

	out.Slides = m.Slides
	dx, dy := s(m.Dest())
	switch {
	case dx == out.X && dy > out.Y:
		out.Type = tak.SlideUp
	case dx == out.X && dy < out.Y:
		out.Type = tak.SlideDown
	case dx < out.X && dy == out.Y:
		out.Type = tak.SlideLeft
	case dx > out.X && dy == out.Y:
		out.Type = tak.SlideRight
	default:
		panic("symmetry is not sane")
	}
	return out
}

func preferMove(l, r tak.Move) bool {
	if l.Y != r.Y {
		return l.Y < r.Y
	}
	if l.X != r.X {
		return l.X < r.X
	}
	return l.Type < r.Type
}

type state struct {
	p     *tak.Position
	s     Symmetry
	moves []tak.Move
}

type PositionAndSymmetry struct {
	P *tak.Position
	S Symmetry
}

func Symmetries(p *tak.Position) ([]PositionAndSymmetry, error) {
	syms := symmetries(p.Size())
	boards := make([][][]tak.Square, len(syms))
	for i := range boards {
		boards[i] = make([][]tak.Square, p.Size())
		for j := range boards[i] {
			boards[i][j] = make([]tak.Square, p.Size())
		}
	}

	for x := 0; x < p.Size(); x++ {
		for y := 0; y < p.Size(); y++ {
			for i, sym := range syms {
				rx, ry := sym(int8(x), int8(y))
				boards[i][ry][rx] = p.At(x, y)
			}
		}
	}

	ps := make([]PositionAndSymmetry, len(boards))
	for i, b := range boards {
		var e error
		ps[i].P, e = tak.FromSquares(p.Config(), b, p.MoveNumber())
		ps[i].S = syms[i]
		if e != nil {
			return nil, e
		}
	}
	seen := make(map[uint64]struct{})
	var out []PositionAndSymmetry
	for _, p := range ps {
		if _, ok := seen[p.P.Hash()]; ok {
			continue
		}
		out = append(out, p)
		seen[p.P.Hash()] = struct{}{}
	}
	return out, nil
}

func Canonical(size int, ms []tak.Move) ([]tak.Move, error) {
	p := tak.New(tak.Config{Size: size})
	syms := symmetries(size)
	boards := make([]*state, len(syms))
	for i := range boards {
		boards[i] = &state{
			s: syms[i],
			p: p,
		}
	}

	var rots []Symmetry
	tfn := syms[0]

	for ply, m := range ms {
		var e error
		h := boards[0].p.Hash()
		m := TransformMove(tfn, m)
		best := m
		var rot Symmetry
		for i, st := range boards {
			if i == 0 {
				continue
			}
			if st.p.Hash() == h {
				rm := TransformMove(st.s, m)
				if preferMove(rm, best) {
					best = rm
					rot = st.s
				}
			}
		}

		if rot != nil {
			rots = append([]Symmetry{rot}, rots...)
			tfn = compose(rots...)
			m = best
		}
		for i, st := range boards {
			rm := TransformMove(st.s, m)
			st.p, e = st.p.Move(rm)
			if e != nil {
				return nil, fmt.Errorf("canonical: move %d: rot %d: %s: %v",
					ply, i, ptn.FormatMove(rm), e)
			}
			st.moves = append(st.moves, rm)
		}
	}

	return boards[0].moves, nil
}
