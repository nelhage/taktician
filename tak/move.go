package tak

import "errors"

type MoveType byte

const (
	PlaceFlat MoveType = 1 + iota
	PlaceStanding
	PlaceCapstone
	SlideLeft
	SlideRight
	SlideUp
	SlideDown
)

const TypeMask MoveType = 0xf

type Move struct {
	X, Y   int
	Type   MoveType
	Slides []byte
}

func (m *Move) Equal(rhs *Move) bool {
	if m.X != rhs.X || m.Y != rhs.Y {
		return false
	}
	if m.Type != rhs.Type {
		return false
	}
	if m.Type < SlideLeft {
		return true
	}
	if len(m.Slides) != len(rhs.Slides) {
		return false
	}
	for i, s := range m.Slides {
		if rhs.Slides[i] != s {
			return false
		}
	}
	return true
}

var (
	ErrOccupied       = errors.New("position is occupied")
	ErrIllegalSlide   = errors.New("illegal slide")
	ErrNoCapstone     = errors.New("capstone has already been played")
	ErrIllegalOpening = errors.New("illegal opening move")
)

func (p *Position) Move(m *Move) (*Position, error) {
	return p.MoveToAllocated(m, nil)
}

func (p *Position) MoveToAllocated(m *Move, next *Position) (*Position, error) {
	if next == nil {
		next = alloc(p)
	} else {
		copyPosition(p, next)
	}
	next.move++
	var place Piece
	dx, dy := 0, 0
	switch m.Type {
	case PlaceFlat:
		place = MakePiece(p.ToMove(), Flat)
	case PlaceStanding:
		place = MakePiece(p.ToMove(), Standing)
	case PlaceCapstone:
		place = MakePiece(p.ToMove(), Capstone)
	case SlideLeft:
		dx = -1
	case SlideRight:
		dx = 1
	case SlideUp:
		dy = 1
	case SlideDown:
		dy = -1
	}
	if p.move < 2 {
		if place.Kind() != Flat {
			return nil, ErrIllegalOpening
		}
		place = MakePiece(place.Color().Flip(), place.Kind())
	}
	i := uint(m.X + m.Y*p.Size())
	if place != 0 {
		if (p.White|p.Black)&(1<<i) != 0 {
			return nil, ErrOccupied
		}

		var stones *byte
		switch place.Kind() {
		case Capstone:
			if p.ToMove() == Black {
				stones = &next.blackCaps
			} else {
				stones = &next.whiteCaps
			}
			next.Caps |= (1 << i)
		case Standing:
			next.Standing |= (1 << i)
			fallthrough
		case Flat:
			if place.Color() == Black {
				stones = &next.blackStones
			} else {
				stones = &next.whiteStones
			}
		}
		if *stones <= 0 {
			return nil, ErrNoCapstone
		}
		*stones--
		if place.Color() == White {
			next.White |= (1 << i)
		} else {
			next.Black |= (1 << i)
		}
		next.Height[i]++
		next.analyze()
		return next, nil
	}

	ct := uint(0)
	for _, c := range m.Slides {
		ct += uint(c)
	}
	if ct > uint(p.cfg.Size) || ct < 1 || ct > uint(p.Height[i]) {
		return nil, ErrIllegalSlide
	}
	if p.ToMove() == White && p.White&(1<<i) == 0 {
		return nil, ErrIllegalSlide
	}
	if p.ToMove() == Black && p.Black&(1<<i) == 0 {
		return nil, ErrIllegalSlide
	}

	top := p.Top(m.X, m.Y)
	stack := p.Stacks[i] << 1
	if top.Color() == Black {
		stack |= 1
	}

	next.Caps &= ^(1 << i)
	next.Standing &= ^(1 << i)
	if uint(next.Height[i]) == ct {
		next.White &= ^(1 << i)
		next.Black &= ^(1 << i)
	} else {
		if stack&(1<<ct) == 0 {
			next.White |= (1 << i)
			next.Black &= ^(1 << i)
		} else {
			next.Black |= (1 << i)
			next.White &= ^(1 << i)
		}
	}
	next.hash ^= next.hashAt(i)
	next.Stacks[i] >>= ct
	next.Height[i] -= uint8(ct)
	next.hash ^= next.hashAt(i)

	x, y := m.X, m.Y
	for _, c := range m.Slides {
		x += dx
		y += dy
		if x < 0 || x >= next.cfg.Size ||
			y < 0 || y >= next.cfg.Size {
			return nil, ErrIllegalSlide
		}
		if int(c) < 1 || uint(c) > ct {
			return nil, ErrIllegalSlide
		}
		i = uint(x + y*p.Size())
		switch {
		case next.Caps&(1<<i) != 0:
			return nil, ErrIllegalSlide
		case next.Standing&(1<<i) != 0:
			if ct != 1 || top.Kind() != Capstone {
				return nil, ErrIllegalSlide
			}
			next.Standing &= ^(1 << i)
		}
		next.hash ^= next.hashAt(i)
		if next.White&(1<<i) != 0 {
			next.Stacks[i] <<= 1
		} else if next.Black&(1<<i) != 0 {
			next.Stacks[i] <<= 1
			next.Stacks[i] |= 1
		}
		drop := (stack >> (ct - uint(c-1))) & ((1 << (c - 1)) - 1)
		next.Stacks[i] = next.Stacks[i]<<(c-1) | drop
		next.Height[i] += c
		next.hash ^= next.hashAt(i)
		if stack&(1<<(ct-uint(c))) != 0 {
			next.Black |= (1 << i)
			next.White &= ^(1 << i)
		} else {
			next.Black &= ^(1 << i)
			next.White |= (1 << i)
		}
		ct -= uint(c)
		if ct == 0 {
			switch top.Kind() {
			case Capstone:
				next.Caps |= (1 << i)
			case Standing:
				next.Standing |= (1 << i)
			}
		}
	}

	next.analyze()
	return next, nil
}

var slides [][][]byte

func init() {
	slides = make([][][]byte, 10)
	for s := 1; s <= 8; s++ {
		slides[s] = calculateSlides(s)
	}
}

func calculateSlides(stack int) [][]byte {
	var out [][]byte
	for i := byte(1); i <= byte(stack); i++ {
		out = append(out, []byte{i})
		for _, sub := range slides[stack-int(i)] {
			t := make([]byte, len(sub)+1)
			t[0] = i
			copy(t[1:], sub)
			out = append(out, t)
		}
	}
	return out
}

func (p *Position) AllMoves() []Move {
	moves := make([]Move, 0, len(p.Stacks))
	next := p.ToMove()
	cap := false
	if next == White {
		cap = p.whiteCaps > 0
	} else {
		cap = p.blackCaps > 0
	}
	for x := 0; x < p.cfg.Size; x++ {
		for y := 0; y < p.cfg.Size; y++ {
			stack := p.At(x, y)
			if len(stack) == 0 {
				moves = append(moves, Move{x, y, PlaceFlat, nil})
				if p.move >= 2 {
					moves = append(moves, Move{x, y, PlaceStanding, nil})
					if cap {
						moves = append(moves, Move{x, y, PlaceCapstone, nil})
					}
				}
				continue
			}
			if p.move < 2 {
				continue
			}
			if stack[0].Color() != next {
				continue
			}
			type dircnt struct {
				d MoveType
				c int
			}
			dirs := [4]dircnt{
				{SlideLeft, x},
				{SlideRight, p.cfg.Size - x - 1},
				{SlideDown, y},
				{SlideUp, p.cfg.Size - y - 1},
			}
			for _, d := range dirs {
				h := len(stack)
				if h > p.cfg.Size {
					h = p.cfg.Size
				}
				for _, s := range slides[h] {
					if len(s) <= d.c {
						moves = append(moves, Move{x, y, d.d, s})
					}
				}
			}
		}
	}

	return moves
}
