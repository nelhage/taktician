package ai

import "nelhage.com/tak/game"

type TakPlayer interface {
	GetMove(p *game.Position) *game.Move
}
