package game

import (
	"errors"
	"log"
)

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

var (
	ErrOccupied     = errors.New("position is occupied")
	ErrIllegalSlide = errors.New("illegal slide")
	ErrNoCapstone   = errors.New("capstone has already been played")
)

func (p *Position) Move(m Move) (*Position, error) {
	var place Piece
	dx, dy := 0, 0
	switch m.Type {
	case PlaceFlat:
		place = makePiece(p.ToMove(), Flat)
	case PlaceStanding:
		place = makePiece(p.ToMove(), Standing)
	case PlaceCapstone:
		place = makePiece(p.ToMove(), Capstone)
	case SlideLeft:
		dx = -1
	case SlideRight:
		dx = 1
	case SlideUp:
		dy = -1
	case SlideDown:
		dy = 1
	}
	next := *p
	next.move++
	if place != 0 {
		if len(p.At(m.X, m.Y)) != 0 {
			return nil, ErrOccupied
		}
		next.board = make([]Square, len(p.board))
		copy(next.board, p.board)
		var stones *byte
		if pieceKind(place) == Capstone {
			if p.ToMove() == Black {
				stones = &next.blackCaps
			} else {
				stones = &next.whiteCaps
			}
		} else {
			if p.ToMove() == Black {
				stones = &next.blackStones
			} else {
				stones = &next.whiteStones
			}
		}
		if *stones == 0 {
			return nil, ErrNoCapstone
		}
		*stones--
		next.set(m.X, m.Y, []Piece{place})
		return &next, nil
	}

	stack := p.At(m.X, m.Y)
	ct := 0
	for _, c := range m.Slides {
		ct += int(c)
	}
	if ct > p.game.Size || ct < 1 || ct > len(stack) {
		log.Printf("illegal size %d", ct)
		return nil, ErrIllegalSlide
	}
	if pieceColor(stack[0]) != p.ToMove() {
		log.Printf("stack not owned")
		return nil, ErrIllegalSlide
	}
	next.board = make([]Square, len(p.board))
	copy(next.board, p.board)
	next.set(m.X, m.Y, stack[ct:])
	stack = stack[:ct]
	for _, c := range m.Slides {
		m.X += dx
		m.Y += dy
		if m.X < 0 || m.X > next.game.Size ||
			m.Y < 0 || m.Y > next.game.Size {
			log.Printf("slide off edge")
			return nil, ErrIllegalSlide
		}
		if int(c) < 1 || int(c) > len(stack) {
			log.Printf("slide empty stack")
			return nil, ErrIllegalSlide
		}
		base := next.At(m.X, m.Y)
		if len(base) > 0 {
			switch pieceKind(base[0]) {
			case Flat:
			case Capstone:
				return nil, ErrIllegalSlide
			case Standing:
				if len(stack) != 1 || pieceKind(stack[0]) != Capstone {
					return nil, ErrIllegalSlide
				}
			}
		}
		tmp := make([]Piece, int(c)+len(base))
		copy(tmp[:c], stack[len(stack)-int(c):])
		copy(tmp[c:], base)
		if len(tmp) > int(c) {
			tmp[c] = makePiece(pieceColor(tmp[c]), Flat)
		}
		next.set(m.X, m.Y, tmp)
		stack = stack[:len(stack)-int(c)]
	}

	return &next, nil
}
