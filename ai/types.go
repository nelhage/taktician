package ai

import (
	"context"

	"github.com/nelhage/taktician/tak"
)

type TakPlayer interface {
	GetMove(ctx context.Context, p *tak.Position) tak.Move
}
