package mcts

import (
	"context"
	"fmt"

	"github.com/nelhage/taktician/bitboard"
	"github.com/nelhage/taktician/tak"
)

type builder func(*MCTSConfig) Policy

var policyMap map[string]builder

func init() {
	policyMap = make(map[string]builder)
	policyMap[""] = buildUniform
	policyMap["uniform"] = buildUniform
}

func (mc *MonteCarloAI) buildPolicy() Policy {
	builder := policyMap[mc.cfg.Policy]
	if builder == nil {
		panic(fmt.Sprintf("no such policy: %s", mc.cfg.Policy))
	}
	return builder(&mc.cfg)
}

type UniformRandom struct {
	alloc *tak.Position
}

func buildUniform(cfg *MCTSConfig) Policy {
	return &UniformRandom{
		alloc: tak.Alloc(cfg.Size),
	}
}

func (u *UniformRandom) Select(ctx context.Context, m *MonteCarloAI, p *tak.Position) *tak.Position {
	moves := p.AllMoves(nil)
	var next *tak.Position
	for {
		r := m.r.Int31n(int32(len(moves)))
		m := moves[r]
		var e error
		if next, e = p.MovePreallocated(m, u.alloc); e == nil {
			break
		}
		moves[0], moves[r] = moves[r], moves[0]
		moves = moves[1:]
	}
	u.alloc = p
	return next
}

type PlaceWins struct {
	uniform UniformRandom
}

func findPlaceWins(mask uint64, empty uint64, gs []uint64, c *bitboard.Constants) uint64 {
	l := mask & c.L
	r := mask & c.R
	b := mask & c.B
	t := mask & c.T
	for _, g := range gs {
		if g&c.L != 0 {
			l |= g
		}
		if g&c.R != 0 {
			r |= g
		}
		if g&c.B != 0 {
			b |= g
		}
		if g&c.T != 0 {
			t |= g
		}
	}
	lf := bitboard.Grow(c, empty, l) | c.L&empty
	rf := bitboard.Grow(c, empty, r) | c.R&empty
	bf := bitboard.Grow(c, empty, b) | c.B&empty
	tf := bitboard.Grow(c, empty, t) | c.T&empty
	/*
		log.Printf("l =%64s", strconv.FormatUint(l, 2))
		log.Printf("lf=%64s", strconv.FormatUint(lf, 2))
		log.Printf("r =%64s", strconv.FormatUint(r, 2))
		log.Printf("rf=%64s", strconv.FormatUint(rf, 2))
		log.Printf("t =%64s", strconv.FormatUint(t, 2))
		log.Printf("tf=%64s", strconv.FormatUint(tf, 2))
		log.Printf("b =%64s", strconv.FormatUint(b, 2))
		log.Printf("bf=%64s", strconv.FormatUint(bf, 2))
	*/
	return (lf & rf) | (tf & bf)
}

func placeWinMove(c *bitboard.Constants, p *tak.Position) tak.Move {
	var myroad uint64
	var gs []uint64
	if p.ToMove() == tak.White {
		myroad = p.White &^ p.Standing
		gs = p.Analysis().WhiteGroups
	} else {
		myroad = p.Black &^ p.Standing
		gs = p.Analysis().BlackGroups
	}
	empty := c.Mask &^ (p.White | p.Black)
	mask := findPlaceWins(myroad, empty, gs, c)
	if mask != 0 {
		bit := mask ^ (mask & (mask - 1))
		x, y := bitboard.BitCoords(c, bit)
		return tak.Move{X: int8(x), Y: int8(y), Type: tak.PlaceFlat}
	}
	return tak.Move{}
}

func (pw *PlaceWins) Select(ctx context.Context, m *MonteCarloAI, p *tak.Position) *tak.Position {
	if move := placeWinMove(&m.c, p); move.Type != 0 {
		out, e := p.MovePreallocated(move, pw.uniform.alloc)
		if e != nil {
			panic("placeWinMove: bad move")
		}
		pw.uniform.alloc = p
		return out
	}
	return pw.uniform.Select(ctx, m, p)
}
