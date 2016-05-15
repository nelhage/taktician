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

	tableSize uint64 = (1 << 20)
)

type EvaluationFunc func(m *MinimaxAI, p *tak.Position) int64

type MinimaxAI struct {
	cfg  MinimaxConfig
	rand *rand.Rand

	st      Stats
	c       bitboard.Constants
	regions []uint64

	evaluate EvaluationFunc

	table []tableEntry
}

type tableEntry struct {
	hash  uint64
	depth int
	value int64
	bound boundType
	m     tak.Move
	p     *tak.Position
}

type boundType byte

const (
	lowerBound = iota
	exactBound = iota
	upperBound = iota
)

type Stats struct {
	Depth     int
	Generated uint64
	Evaluated uint64
	Cutoffs   uint64
	TTHits    uint64
}

type MinimaxConfig struct {
	Size  int
	Depth int
	Debug int
	Seed  int64

	Evaluate EvaluationFunc
}

func NewMinimax(cfg MinimaxConfig) *MinimaxAI {
	m := &MinimaxAI{cfg: cfg}
	m.precompute()
	m.evaluate = cfg.Evaluate
	if m.evaluate == nil {
		m.evaluate = DefaultEvaluate
	}
	m.table = make([]tableEntry, tableSize)
	return m
}

func (m *MinimaxAI) ttGet(h uint64) *tableEntry {
	te := &m.table[h%tableSize]
	if te.hash != h {
		return nil
	}
	return te
}

func (m *MinimaxAI) ttPut(t *tableEntry) {
	m.table[t.hash%tableSize] = *t
}

func (m *MinimaxAI) precompute() {
	s := uint(m.cfg.Size)
	m.c = bitboard.Precompute(s)
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
	base := 0
	te := m.ttGet(p.Hash())
	if te != nil && te.bound == exactBound {
		base = te.depth
		ms = []tak.Move{te.m}
	}

	for i := 1; i+base <= m.cfg.Depth; i++ {
		m.st = Stats{Depth: i + base}
		start := time.Now()
		ms, v = m.minimax(p, 0, i+base, ms, minEval-1, maxEval+1)
		timeUsed := time.Now().Sub(top)
		timeMove := time.Now().Sub(start)
		if m.cfg.Debug > 0 {
			log.Printf("[minimax] deepen: depth=%d val=%d pv=%s time=%s total=%s evaluated=%d tt=%d branch=%d",
				base+i, v, formatpv(ms),
				timeMove,
				timeUsed,
				m.st.Evaluated,
				m.st.TTHits,
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
		if i > 2 && i != m.cfg.Depth && limit != 0 {
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

const debugTable = false

func (ai *MinimaxAI) minimax(
	p *tak.Position,
	ply, depth int,
	pv []tak.Move,
	α, β int64) ([]tak.Move, int64) {
	over, _ := p.GameOver()
	if depth == 0 || over {
		ai.st.Evaluated++
		return nil, ai.evaluate(ai, p)
	}

	moves := p.AllMoves()
	ai.st.Generated += uint64(len(moves))
	if ply == 0 {
		for i := len(moves) - 1; i > 0; i-- {
			j := ai.rand.Int31n(int32(i))
			moves[j], moves[i] = moves[i], moves[j]
		}
	}
	te := ai.ttGet(p.Hash())
	if te != nil {
		if te.depth >= depth {
			if te.bound == exactBound ||
				(te.value < α && te.bound == upperBound) ||
				(te.value > β && te.bound == lowerBound) {
				ai.st.TTHits++
				return []tak.Move{te.m}, te.value
			}
		}

		if te.bound == exactBound &&
			(te.value > winThreshold || te.value < -winThreshold) {
			ai.st.TTHits++
			return []tak.Move{te.m}, te.value
		}
	}
	if len(pv) > 0 {
		j := 1
		for i, m := range moves {
			if m.Equal(&pv[0]) {
				moves[0], moves[i] = moves[i], moves[0]
				if m.Type < tak.SlideLeft {
					break
				}

			} else if te != nil && j < len(moves) && te.m.Equal(&m) {
				moves[j], moves[i] = moves[i], moves[j]
				j++
			} else if j < len(moves) && m.X == pv[0].X && m.Y == pv[0].Y {
				moves[j], moves[i] = moves[i], moves[j]
				j++
			}
		}
	}

	best := make([]tak.Move, 0, depth)
	best = append(best, pv...)
	improved := false
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

		if len(best) == 0 {
			best = append(best[:0], m)
			best = append(best, ms...)
		}
		if v > α {
			improved = true
			best = append(best[:0], m)
			best = append(best, ms...)
			α = v
			if α >= β {
				ai.st.Cutoffs++
				break
			}
		}
	}

	if debugTable && te != nil &&
		te.depth == depth &&
		te.bound == exactBound &&
		!best[0].Equal(&te.m) {
		log.Printf("? ply=%d depth=%d found=[%s, %v] t=[%s, %v]",
			ply, depth,
			ptn.FormatMove(&best[0]), α,
			ptn.FormatMove(&te.m), te.value,
		)
		log.Printf(" p> %#v", p)
		log.Printf("tp> %#v", te.p)
	}

	if depth > 1 {
		te := tableEntry{
			hash:  p.Hash(),
			depth: depth,
			m:     best[0],
			value: α,
		}
		if debugTable {
			te.p = p
		}
		if !improved {
			te.bound = upperBound
		} else if α >= β {
			te.bound = lowerBound
		} else {
			te.bound = exactBound
		}
		ai.ttPut(&te)
	}

	return best, α
}
