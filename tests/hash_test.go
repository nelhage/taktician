package tests

import (
	"flag"
	"testing"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

var hashTests = flag.Bool("test-hash", false, "run hash collision tests")

func wrapHash(tbl map[uint64][]*tak.Position, eval ai.EvaluationFunc) ai.EvaluationFunc {
	return func(m *ai.MinimaxAI, p *tak.Position) int64 {
		tbl[p.Hash()] = append(tbl[p.Hash()], p)
		return eval(m, p)
	}
}

func equal(a, b *tak.Position) bool {
	if a.White != b.White {
		return false
	}
	if a.Black != b.Black {
		return false
	}
	if a.Standing != b.Standing {
		return false
	}
	if a.Caps != b.Caps {
		return false
	}
	for i := range a.Height {
		if a.Height[i] != b.Height[i] {
			return false
		}
		if a.Stacks[i] != b.Stacks[i] {
			return false
		}
	}
	return true
}

func reportCollisions(t *testing.T, tbl map[uint64][]*tak.Position) {
	var n, collisions int
	for h, l := range tbl {
		n += len(l)
		p := l[0]
		for _, pp := range l[1:] {
			if !equal(p, pp) {
				t.Logf(" collision h=%x l=%q r=%q",
					h, ptn.FormatTPS(p), ptn.FormatTPS(pp),
				)
				collisions++
				break
			}
		}
	}

	t.Logf("evaluated %d positions and %d hashes, with %d collisions",
		n, len(tbl), collisions)
}

func TestHash(t *testing.T) {
	tbl := make(map[uint64][]*tak.Position)
	if !*hashTests {
		t.SkipNow()
	}
	p := tak.New(tak.Config{Size: 5})
	ai := ai.NewMinimax(ai.MinimaxConfig{
		Size:     5,
		Depth:    6,
		Evaluate: wrapHash(tbl, ai.DefaultEvaluate),
		NoTable:  true,
	})
	ai.GetMove(p, 0)
	reportCollisions(t, tbl)
}
