package ai

import "github.com/nelhage/taktician/tak"

type TakPlayer interface {
	GetMove(p *tak.Position) tak.Move
}
