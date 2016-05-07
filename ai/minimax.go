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
	depth int
	size  uint
	rand  *rand.Rand

	Seed  int64
	Debug int

	st      Stats
	c       bitboard.Constants
	regions []uint64
	rd      int
}

type Stats struct {
	Generated uint64
	Evaluated uint64
	Cutoffs   uint64
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
	if m.size != uint(p.Size()) {
		panic("Analyze: wrong size")
	}

	var seed int64
	if m.Seed == 0 {
		seed = time.Now().Unix()
	} else {
		seed = m.Seed
	}
	m.rand = rand.New(rand.NewSource(seed))
	if m.Debug > 0 {
		log.Printf("seed=%d", seed)
	}

	var ms []tak.Move
	var v int64
	top := time.Now()
	var prevEval uint64
	var branchSum uint64
	for i := 1; i <= m.depth; i++ {
		m.st = Stats{}
		start := time.Now()
		ms, v = m.minimax(p, 0, i, ms, minEval-1, maxEval+1)
		timeUsed := time.Now().Sub(top)
		timeMove := time.Now().Sub(start)
		if m.Debug > 0 {
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
		if i > 2 && i != m.depth {
			estimate := timeUsed + time.Now().Sub(start)*time.Duration(branchSum/uint64(i-1))
			if estimate > limit {
				if m.Debug > 0 {
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
		if ai.Debug > 2 && ply == 0 {
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

func imin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func imax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func iabs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

const (
	weightFlat       = 200
	weightCaptured   = 100
	weightControlled = 500
	weightCapstone   = -150
	weightThreat     = 150
	weightAdvantage  = 50
)

func (m *MinimaxAI) evaluate(p *tak.Position) int64 {
	if over, winner := p.GameOver(); over {
		switch winner {
		case tak.NoColor:
			return 0
		case p.ToMove():
			return maxEval - int64(p.MoveNumber())
		default:
			return minEval + int64(p.MoveNumber())
		}
	}
	mine, theirs := 0, 0
	me := p.ToMove()
	addw := func(c tak.Color, w int) {
		if c == me {
			mine += w
		} else {
			theirs += w
		}
	}
	analysis := p.Analysis()
	for x := 0; x < p.Size(); x++ {
		for y := 0; y < p.Size(); y++ {
			sq := p.At(x, y)
			if len(sq) == 0 {
				continue
			}
			addw(sq[0].Color(), weightControlled)
			if sq[0].Kind() == tak.Capstone {
				addw(sq[0].Color(), weightCapstone)
			}

			for i, stone := range sq {
				if i > 0 && i < p.Size() {
					addw(sq[0].Color(), weightCaptured)
				}
				if stone.Kind() == tak.Flat {
					addw(stone.Color(), weightFlat)
				}
			}
		}
	}
	for _, r := range m.regions {
		w := bitboard.Popcount(analysis.White & r)
		b := bitboard.Popcount(analysis.Black & r)
		if w > b {
			addw(tak.White, (w-b)*weightAdvantage)
		} else {
			addw(tak.Black, (b-w)*weightAdvantage)
		}
	}
	o := analysis.White | analysis.Black
	addw(tak.White, weightThreat*m.threats(analysis.WhiteGroups, o))
	addw(tak.Black, weightThreat*m.threats(analysis.BlackGroups, o))

	return int64(mine - theirs)
}

func (m *MinimaxAI) threats(groups []uint64, filled uint64) int {
	count := 0
	empty := ^filled
	s := m.size
	for _, g := range groups {
		if g&m.c.L != 0 {
			if g&(m.c.R<<1) != 0 && empty&m.c.R != 0 {
				count++
			}
		}
		if g&m.c.R != 0 {
			if g&(m.c.L>>1) != 0 && empty&m.c.L != 0 {
				count++
			}
		}
		if g&m.c.B != 0 {
			if g&(m.c.T>>s) != 0 && empty&m.c.T != 0 {
				count++
			}
		}
		if g&m.c.T != 0 {
			if g&(m.c.B<<s) != 0 && empty&m.c.B != 0 {
				count++
			}
		}
	}
	return count
}

func (m *MinimaxAI) precompute() {
	m.c = bitboard.Precompute(m.size)
	if m.size == 5 { // TODO(board-size)
		br := uint64((1 << 3) - 1)
		br |= br<<m.size | br<<(2*m.size)
		m.regions = []uint64{
			br, br << 2,
			br << (2 * m.size), br << (2*m.size + 2),
		}
		m.rd = 2
	}
}

func NewMinimax(size int, depth int) *MinimaxAI {
	m := &MinimaxAI{size: uint(size), depth: depth}
	m.precompute()
	return m
}
