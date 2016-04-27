package ai

import "nelhage.com/tak/tak"

type TakPlayer interface {
	GetMove(p *tak.Position) tak.Move
}
