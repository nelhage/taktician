package main

import (
	"errors"

	"github.com/nelhage/taktician/tak"
)

type FPARule interface {
	Greeting(tak.Color) string
	LegalMove(p *tak.Position, m tak.Move) error
	GetMove(p *tak.Position) (tak.Move, bool)
}

type CenterBlack struct{}

func (c *CenterBlack) Greeting(color tak.Color) string {
	if color == tak.Black {
		return "I'm an experiment bot testing alternate rules. " +
			"To play me, please place my first stone " +
			"in the center of the board."
	}
	return ""
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
