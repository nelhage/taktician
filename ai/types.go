package ai

import (
	"time"

	"github.com/nelhage/taktician/tak"
)

type TakPlayer interface {
	GetMove(p *tak.Position, limit time.Duration) tak.Move
}
