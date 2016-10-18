package canonicalize

import (
	"fmt"

	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

type symmetry func(int, int) (int, int)

func compose(ss ...symmetry) symmetry {
	return func(x, y int) (int, int) {
		for i := range ss {
			s := ss[len(ss)-i-1]
			x, y = s(x, y)
		}
		return x, y
	}
}

func symmetries(size int) []symmetry {
	flip := func(i int) int {
		return size - 1 - i
	}

	identity := func(x, y int) (int, int) {
		return x, y
	}

	flipX := func(x, y int) (int, int) {
		return flip(x), y
	}

	flipY := func(x, y int) (int, int) {
		return x, flip(y)
	}
	flipDiag1 := func(x, y int) (int, int) {
		return y, x
	}
	flipDiag2 := func(x, y int) (int, int) {
		return flip(y), flip(x)
	}

	rotate := func(x, y int) (int, int) {
		return flip(x), flip(y)
	}

	return []symmetry{
		identity,
		flipX,
		flipY,
		flipDiag1,
		flipDiag2,
		rotate,
	}
}

func rotateMove(s symmetry, m *tak.Move) tak.Move {
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

func preferMove(l, r *tak.Move) bool {
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
	s     symmetry
	moves []tak.Move
}

func Canonical(size int, ms []tak.Move) []tak.Move {
	p := tak.New(tak.Config{Size: size})
	syms := symmetries(size)
	boards := make([]*state, len(syms))
	for i := range boards {
		boards[i] = &state{
			s: syms[i],
			p: p,
		}
	}

	var rots []symmetry
	tfn := syms[0]

	for _, m := range ms {
		var e error
		h := boards[0].p.Hash()
		best := m
		var rot symmetry
		for i, st := range boards {
			if i == 0 {
				continue
			}
			if st.p.Hash() == h {
				rm := rotateMove(st.s, &m)
				if preferMove(&rm, &best) {
					best = rm
					rot = st.s
				}
			}
		}

		if rot != nil {
			rots = append([]symmetry{rot}, rots...)
			tfn = compose(rots...)
		}
		for i, st := range boards {
			rm := rotateMove(tfn, &m)
			rm = rotateMove(st.s, &rm)
			st.p, e = st.p.Move(&rm)
			if e != nil {
				panic(fmt.Sprintf("[%d] move %#v->%#v(%s): %v",
					i, &m, rm, ptn.FormatMove(&rm), e))
			}
			st.moves = append(st.moves, rm)
		}
	}

	return boards[0].moves
}
