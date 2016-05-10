package ai

import (
	"bytes"
	"log"
	"math/rand"
	"sync"
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
	Depth     int
	Generated uint64
	Evaluated uint64
	Cutoffs   uint64
}

type MinimaxConfig struct {
	Size  int
	Depth int
	Debug int
	Seed  int64
	Spawn int
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
	if m.cfg.Spawn == 0 {
		m.cfg.Spawn = 1
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
		m.st = Stats{Depth: i}
		start := time.Now()
		ms, v = m.minimax(p, 0, i, ms, minEval-1, maxEval+1)
		timeUsed := time.Now().Sub(top)
		timeMove := time.Now().Sub(start)
		if m.cfg.Debug > 0 {
			log.Printf("[minimax] deepen: depth=%d val=%d pv=%s time=%s total=%s evaluated=%d cutoffs=%d branch=%d",
				i, v, formatpv(ms),
				timeMove,
				timeUsed,
				m.st.Evaluated,
				m.st.Cutoffs,
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

func (ai *MinimaxAI) order(moves []tak.Move, ply int, pv []tak.Move) {
	if ply == 0 {
		for i := len(moves) - 1; i > 0; i-- {
			j := ai.rand.Int31n(int32(i))
			moves[j], moves[i] = moves[i], moves[j]
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
			} else if j < len(moves) && m.X == pv[0].X && m.Y == pv[0].Y {
				moves[j], moves[i] = moves[i], moves[j]
				j++
			}
		}
	}
}

type mmJob struct {
	p          *tak.Position
	ply, depth int

	sync.Mutex
	α, β int64
	best []tak.Move

	ms <-chan tak.Move
	wg sync.WaitGroup
}

func (ai *MinimaxAI) mmWorker(j *mmJob) {
	defer j.wg.Done()

	pv := make([]tak.Move, 0, j.depth)
	for m := range j.ms {
		child, e := j.p.Move(&m)
		if e != nil {
			continue
		}
		j.Lock()
		if len(j.best) != 0 {
			pv = append(pv[:0], j.best[1:]...)
		}
		α, β := j.α, j.β
		j.Unlock()
		if α >= β {
			ai.st.Cutoffs++
			break
		}
		ms, v := ai.minimax(child, j.ply+1, j.depth-1, pv, -β, -α)
		v = -v
		if v > α {
			j.Lock()
			if v > j.α {
				j.best = append(j.best[:0], m)
				j.best = append(j.best, ms...)
				j.α = v
			}
			j.Unlock()
		}
	}
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

	moves := p.AllMoves()
	ai.st.Generated += uint64(len(moves))
	ai.order(moves, ply, pv)

	best := make([]tak.Move, 0, depth)
	best = append(best, pv...)

	i := 0
	var m tak.Move
	var ms []tak.Move
	var v int64

	spawn := ai.cfg.Spawn
	if depth < 5 {
		spawn = 1
	}

	for i, m = range moves {
		child, e := p.Move(&m)
		if e != nil {
			continue
		}
		var newpv []tak.Move
		if len(best) != 0 {
			newpv = best[1:]
		}
		ms, v = ai.minimax(child, ply+1, depth-1, newpv, -β, -α)
		v = -v
		if len(best) == 0 {
			best = append(best[:0], m)
			best = append(best, ms...)
		}
		if v > α {
			best = append(best[:0], m)
			best = append(best, ms...)
			α = v
			if α >= β {
				ai.st.Cutoffs++
				return best, v
			}
		}
		if spawn > 1 {
			break
		}
	}
	mc := make(chan tak.Move, spawn)
	job := &mmJob{
		p: p,
		α: α, β: β,
		best:  best,
		ms:    mc,
		depth: depth,
		ply:   ply,
	}
	if i < len(moves) {
		job.wg.Add(spawn)
		for i := 0; i < spawn; i++ {
			go ai.mmWorker(job)
		}
		done := make(chan struct{})
		go func() {
			job.wg.Wait()
			close(done)
		}()
	outer:
		for _, m := range moves[i:] {
			select {
			case mc <- m:
			case <-done:
				break outer
			}
		}
		close(mc)
		<-done
	}
	return job.best, job.α
}
