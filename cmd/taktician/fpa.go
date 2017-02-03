package main

import (
	"errors"

	"github.com/nelhage/taktician/tak"
)

type FPARule interface {
	Greeting(tak.Color) []string
	LegalMove(p *tak.Position, m tak.Move) error
	GetMove(p *tak.Position) (tak.Move, bool)
}

type CenterBlack struct{}

func (c *CenterBlack) Greeting(color tak.Color) []string {
	if color == tak.White {
		return nil
	}
	return []string{
		"I'm an experiment bot testing alternate rules. " +
			"To play me, please place my first stone " +
			"in the center of the board."}
}

func (c *CenterBlack) LegalMove(p *tak.Position, m tak.Move) error {
	if p.MoveNumber() > 0 {
		return nil
	}
	if c.isCentered(p, m) {
		return nil
	}
	return errors.New("I'm testing rules to balance white's advantage. " +
		"To play me as white, you must place Black's first " +
		"piece in the center of the board.")
}

func (c *CenterBlack) GetMove(p *tak.Position) (tak.Move, bool) {
	if p.MoveNumber() > 0 {
		return tak.Move{}, false
	}
	return tak.Move{
		X: int8(p.Size() / 2), Y: int8(p.Size() / 2),
		Type: tak.PlaceFlat,
	}, true
}

func (c *CenterBlack) isCentered(p *tak.Position, m tak.Move) bool {
	mid := int8(p.Size() / 2)
	if p.Size()%2 == 1 {
		// Must be in exact center
		return m.X == mid && m.Y == mid
	}
	return (m.X == mid || m.X == mid-1) &&
		(m.Y == mid || m.Y == mid-1)
}

type DoubleStack struct {
	blackPlace tak.Move
	blackTmp   struct {
		x, y int
	}
	whitePlace tak.Move
	whiteTmp   struct {
		x, y int
	}
}

func (d *DoubleStack) Greeting(tak.Color) []string {
	return []string{
		"I'm an experimental bot testing alternate rules (Double Black Stack). " +
			"Black's very first stone will be a double stack instead of a normal piece. ",
		"Place the very first white and black stone normally. " +
			"Then, white should waste 2 moves moving back and forth. " +
			"Black should place a flat and stack on top of the original location. " +
			"Then, the game continues normally. ",
		"For an example opening, see https://goo.gl/RpLxhe",
	}
}

var doubleStackErrors = []string{
	"",
	"",

	"As white, you needed to move your original piece 1 square, " +
		"wasting time so black can create a double stack " +
		"(Double Black Stack FPA experiment)",

	"As black, you needed to place a flat adjacent to your original piece, " +
		"so you can create a double stack next move. " +
		"(Double Black Stack FPA experiment)",

	"As white, you needed to move your original piece back to where it started, " +
		"wasting time so black can create a double stack " +
		"(Double Black Stack FPA experiment)",

	"As black, you needed to create a double stack on your original piece. " +
		"(Double Black Stack FPA experiment)",
}

func (d *DoubleStack) LegalMove(p *tak.Position, m tak.Move) error {
	ok := true
	switch p.MoveNumber() {
	case 0:
		// White places black anywhere
		d.blackPlace = m
	case 1:
		// Black places black anywhere
		d.whitePlace = m
	case 2:
		// White slides
		ok = m.IsSlide()
		ex, ey := m.Dest()
		d.whiteTmp.x, d.whiteTmp.y = int(ex), int(ey)
	case 3:
		// Black places adjacent to their first piece
		if m.Type != tak.PlaceFlat {
			ok = false
			break
		}
		dx := m.X - d.blackPlace.X
		dy := m.Y - d.blackPlace.Y
		if dx < 0 {
			dx = -dx
		}
		if dy < 0 {
			dy = -dy
		}
		ok = (dx == 1 && dy == 0) || (dx == 0 && dy == 1)
		d.blackTmp.x, d.blackTmp.y = int(m.X), int(m.Y)
	case 4:
		// White slides back
		if !m.IsSlide() {
			ok = false
			break
		}
		ex, ey := m.Dest()
		ok = (ex == d.whitePlace.X && ey == d.whitePlace.Y)
	case 5:
		// Black stacks
		if !m.IsSlide() {
			ok = false
			break
		}
		ex, ey := m.Dest()
		ok = (ex == d.blackPlace.X && ey == d.blackPlace.Y)
	default:
	}
	if ok {
		return nil
	}
	return errors.New(doubleStackErrors[p.MoveNumber()])
}

func dir(x, y, ex, ey int) tak.MoveType {
	switch {
	case x < ex:
		return tak.SlideRight
	case x > ex:
		return tak.SlideLeft
	case y < ey:
		return tak.SlideUp
	case y > ey:
		return tak.SlideDown
	}
	panic("bad dir() call")
}

func adjacent(p *tak.Position, x, y int) (int, int) {
	switch {
	case x > 0 && p.Top(x-1, y) == 0:
		return x - 1, y
	case y > 0 && p.Top(x, y-1) == 0:
		return x, y - 1
	case x+1 < p.Size() && p.Top(x+1, y) == 0:
		return x + 1, y
	case y+1 < p.Size() && p.Top(x, y+1) == 0:
		return x, y + 1
	}
	panic("no empty adjacency")
}

func (d *DoubleStack) GetMove(p *tak.Position) (tak.Move, bool) {
	switch p.MoveNumber() {
	case 0, 1:
		return tak.Move{}, false
	case 2:
		// White slides
		x, y := int(d.whitePlace.X), int(d.whitePlace.Y)
		ex, ey := adjacent(p, x, y)
		m := tak.Move{
			X:      d.whitePlace.X,
			Y:      d.whitePlace.Y,
			Type:   dir(x, y, ex, ey),
			Slides: tak.MkSlides(1),
		}
		return m, true
	case 3:
		// Black places adjacent to their first piece
		x, y := int(d.blackPlace.X), int(d.blackPlace.Y)
		ex, ey := adjacent(p, x, y)
		m := tak.Move{
			X:    int8(ex),
			Y:    int8(ey),
			Type: tak.PlaceFlat,
		}
		return m, true
	case 4:
		// White slides back
		m := tak.Move{
			X: int8(d.whiteTmp.x),
			Y: int8(d.whiteTmp.y),
			Type: dir(d.whiteTmp.x, d.whiteTmp.y,
				int(d.whitePlace.X), int(d.whitePlace.Y)),
			Slides: tak.MkSlides(1),
		}
		return m, true
	case 5:
		// Black stacks
		m := tak.Move{
			X: int8(d.blackTmp.x),
			Y: int8(d.blackTmp.y),
			Type: dir(d.blackTmp.x, d.blackTmp.y,
				int(d.blackPlace.X), int(d.blackPlace.Y)),
			Slides: tak.MkSlides(1),
		}
		return m, true
	default:
		return tak.Move{}, false
	}
}
