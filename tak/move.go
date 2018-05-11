package tak

import "errors"

//go:generate stringer -type=MoveType
type MoveType byte

const (
	Pass MoveType = 1 + iota
	PlaceFlat
	PlaceStanding
	PlaceCapstone
	SlideLeft
	SlideRight
	SlideUp
	SlideDown
)

const TypeMask MoveType = 0xf

type Move struct {
	X, Y   int8
	Type   MoveType
	Slides Slides
}

func (m Move) Equal(rhs Move) bool {
	if m.X != rhs.X || m.Y != rhs.Y {
		return false
	}
	if m.Type != rhs.Type {
		return false
	}
	if !m.IsSlide() {
		return true
	}
	if m.Slides != rhs.Slides {
		return false
	}
	return true
}

func (m Move) IsSlide() bool {
	return m.Type >= SlideLeft
}

func (m Move) Dest() (int8, int8) {
	switch m.Type {
	case PlaceFlat, PlaceStanding, PlaceCapstone:
		return m.X, m.Y
	case SlideLeft:
		return m.X - int8(m.Slides.Len()), m.Y
	case SlideRight:
		return m.X + int8(m.Slides.Len()), m.Y
	case SlideUp:
		return m.X, m.Y + int8(m.Slides.Len())
	case SlideDown:
		return m.X, m.Y - int8(m.Slides.Len())
	}
	panic("bad type")
}

var (
	ErrOccupied       = errors.New("position is occupied")
	ErrIllegalSlide   = errors.New("illegal slide")
	ErrNoCapstone     = errors.New("capstone has already been played")
	ErrIllegalOpening = errors.New("illegal opening move")
)

func (p *Position) Move(m Move) (*Position, error) {
	return p.MovePreallocated(m, nil)
}

func (p *Position) MovePreallocated(m Move, next *Position) (*Position, error) {
	if next == nil {
		next = alloc(p)
	} else {
		copyPosition(p, next)
	}
	next.move++
	var place Piece
	var dx, dy int8
	switch m.Type {
	case Pass:
		next.analyze()
		return next, nil
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
	default:
		return nil, errors.New("invalid move type")
	}
	if p.move < 2 {
		if place.Kind() != Flat {
			return nil, ErrIllegalOpening
		}
		place = MakePiece(place.Color().Flip(), place.Kind())
	}
	i := uint(m.X + m.Y*int8(p.Size()))
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
	for it := m.Slides.Iterator(); it.Ok(); it = it.Next() {
		c := it.Elem()
		if c == 0 {
			return nil, ErrIllegalSlide
		}
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

	top := p.Top(int(m.X), int(m.Y))
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
	for it := m.Slides.Iterator(); it.Ok(); it = it.Next() {
		c := uint(it.Elem())
		x += dx
		y += dy
		if x < 0 || x >= int8(next.cfg.Size) ||
			y < 0 || y >= int8(next.cfg.Size) {
			return nil, ErrIllegalSlide
		}
		if int(c) < 1 || uint(c) > ct {
			return nil, ErrIllegalSlide
		}
		i = uint(x + y*int8(p.Size()))
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
		next.Height[i] += uint8(c)
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

var slides [][]Slides

func init() {
	slides = make([][]Slides, 10)
	for s := 1; s <= 8; s++ {
		slides[s] = calculateSlides(s)
	}
}

func calculateSlides(stack int) []Slides {
	var out []Slides
	for i := byte(1); i <= byte(stack); i++ {
		out = append(out, MkSlides(int(i)))
		for _, sub := range slides[stack-int(i)] {
			out = append(out, sub.Prepend(int(i)))
		}
	}
	return out
}

func (p *Position) AllMoves(moves []Move) []Move {
	next := p.ToMove()
	cap := false
	if next == White {
		cap = p.whiteCaps > 0
	} else {
		cap = p.blackCaps > 0
	}

	for x := 0; x < p.cfg.Size; x++ {
		for y := 0; y < p.cfg.Size; y++ {
			i := uint(y*p.cfg.Size + x)
			if p.Height[i] == 0 {
				moves = append(moves, Move{X: int8(x), Y: int8(y), Type: PlaceFlat})
				if p.move >= 2 {
					moves = append(moves, Move{X: int8(x), Y: int8(y), Type: PlaceStanding})
					if cap {
						moves = append(moves, Move{X: int8(x), Y: int8(y), Type: PlaceCapstone})
					}
				}
				continue
			}
			if p.move < 2 {
				continue
			}
			if next == White && p.White&(1<<i) == 0 {
				continue
			} else if next == Black && p.Black&(1<<i) == 0 {
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
				h := p.Height[i]
				if h > uint8(p.cfg.Size) {
					h = uint8(p.cfg.Size)
				}
				mask := ^Slides((1 << (4 * uint(d.c))) - 1)
				for _, s := range slides[h] {
					if s&mask == 0 {
						moves = append(moves, Move{X: int8(x), Y: int8(y), Type: d.d, Slides: s})
					}
				}
			}
		}
	}

	return moves
}
