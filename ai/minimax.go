package ai

import (
	"bytes"
	"log"
	"math/rand"
	"time"

	"github.com/nelhage/taktician/bitboard"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

const (
	maxEval      int64 = 1 << 30
	minEval            = -maxEval
	winThreshold       = 1 << 29
)

type MinimaxAI struct {
	cfg  MinimaxConfig
	rand *rand.Rand

	st      Stats
	c       bitboard.Constants
	regions []uint64
}

type Stats struct {
	Generated uint64
	Evaluated uint64
	Cutoffs   uint64
}

type MinimaxConfig struct {
	Size  int
	Depth int
	Debug int
	Seed  int64
}

func NewMinimax(cfg MinimaxConfig) *MinimaxAI {
	m := &MinimaxAI{cfg: cfg}
	m.precompute()
	return m
}

func (m *MinimaxAI) precompute() {
	s := uint(m.cfg.Size)
	m.c = bitboard.Precompute(s)
	switch m.cfg.Size {
	// TODO(board-size)
	case 5:
		br := uint64((1 << 3) - 1)
		br |= br<<s | br<<(2*s)
		m.regions = []uint64{
			br, br << 2,
			br << (2 * s), br << (2*s + 2),
		}
	case 6:
		br := uint64((1 << 3) - 1)
		br |= br<<s | br<<(2*s)
		m.regions = []uint64{
			br, br << 3,
			br << (3 * s), br << (3*s + 3),
		}
	}
}

func formatpv(ms []tak.Move) string {
	var out bytes.Buffer
	out.WriteString("[")
	for i, m := range ms {
		if i != 0 {
			out.WriteString(" ")
		}
		out.WriteString(ptn.FormatMove(&m))
	}
	out.WriteString("]")
	return out.String()
}

func (m *MinimaxAI) GetMove(p *tak.Position, limit time.Duration) tak.Move {
	ms, _, _ := m.Analyze(p, limit)
	return ms[0]
}

func (m *MinimaxAI) Analyze(p *tak.Position, limit time.Duration) ([]tak.Move, int64, Stats) {
	if m.cfg.Size != p.Size() {
		panic("Analyze: wrong size")
	}

	var seed = m.cfg.Seed
	if seed == 0 {
		seed = time.Now().Unix()
	}
	m.rand = rand.New(rand.NewSource(seed))
	if m.cfg.Debug > 0 {
		log.Printf("seed=%d", seed)
	}

	var ms []tak.Move
	var v int64
	top := time.Now()
	var prevEval uint64
	var branchSum uint64
	for i := 1; i <= m.cfg.Depth; i++ {
		m.st = Stats{}
		start := time.Now()
		ms, v = m.minimax(p, 0, i, ms, minEval-1, maxEval+1)
		timeUsed := time.Now().Sub(top)
		timeMove := time.Now().Sub(start)
		if m.cfg.Debug > 0 {
			log.Printf("[minimax] deepen: depth=%d val=%d pv=%s time=%s total=%s evaluated=%d branch=%d",
				i, v, formatpv(ms),
				timeMove,
				timeUsed,
				m.st.Evaluated,
				m.st.Evaluated/(prevEval+1),
			)
		}
		if i > 1 {
			branchSum += m.st.Evaluated / (prevEval + 1)
		}
		prevEval = m.st.Evaluated
		if v > winThreshold || v < -winThreshold {
			break
		}
		if i > 2 && i != m.cfg.Depth {
			estimate := timeUsed + time.Now().Sub(start)*time.Duration(branchSum/uint64(i-1))
			if estimate > limit {
				if m.cfg.Debug > 0 {
					log.Printf("[minimax] time cutoff: depth=%d used=%s estimate=%s",
						i, timeUsed, estimate)
				}
				break
			}
		}
	}
	return ms, v, m.st
}

func (ai *MinimaxAI) minimax(
	p *tak.Position,
	ply, depth int,
	pv []tak.Move,
	α, β int64) ([]tak.Move, int64) {
	over, _ := p.GameOver()
	if depth == 0 || over {
		ai.st.Evaluated++
		return nil, ai.evaluate(p)
	}

	if p.MoveNumber() < 2 {
		for _, c := range [][]int{{0, 0}, {p.Size() - 1, 0}, {0, p.Size() - 1}, {p.Size() - 1, p.Size() - 1}} {
			x, y := c[0], c[1]
			if len(p.At(x, y)) == 0 {
				return []tak.Move{{X: x, Y: y, Type: tak.PlaceFlat}}, 0
			}
		}
	}
	moves := p.AllMoves()
	ai.st.Generated += uint64(len(moves))
	if ply == 0 {
		for i := len(moves) - 1; i > 0; i-- {
			j := ai.rand.Int31n(int32(i))
			moves[j], moves[i] = moves[i], moves[j]
		}
	}
	if len(pv) > 0 {
		for i, m := range moves {
			if m.Equal(&pv[0]) {
				moves[0], moves[i] = moves[i], moves[0]
				break
			}
		}
	}

	best := make([]tak.Move, 0, depth)
	best = append(best, pv...)
	max := minEval - 1
	for _, m := range moves {
		child, e := p.Move(&m)
		if e != nil {
			continue
		}
		var ms []tak.Move
		var newpv []tak.Move
		var v int64
		if len(best) != 0 {
			newpv = best[1:]
		}
		ms, v = ai.minimax(child, ply+1, depth-1, newpv, -β, -α)
		v = -v
		if ai.cfg.Debug > 2 && ply == 0 {
			log.Printf("[minimax] search: depth=%d ply=%d m=%s pv=%s window=(%d,%d) ms=%s v=%d evaluated=%d",
				depth, ply, ptn.FormatMove(&m), formatpv(newpv), α, β, formatpv(ms), v, ai.st.Evaluated)
		}
		if v > max {
			max = v
			best = append(best[:0], m)
			best = append(best, ms...)
		}
		if v > α {
			α = v
			if α >= β {
				ai.st.Cutoffs++
				break
			}
		}
	}
	return best, max
}
