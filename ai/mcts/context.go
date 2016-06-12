package mcts

import (
	"math/rand"

	"golang.org/x/net/context"
)

type key int

var randKey key

func WithRand(ctx context.Context, r *rand.Rand) context.Context {
	return context.WithValue(ctx, randKey, r)
}

func GetRand(ctx context.Context) *rand.Rand {
	r, _ := ctx.Value(randKey).(*rand.Rand)
	return r
}
