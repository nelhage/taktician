package ai

import (
	"sort"

	"github.com/nelhage/taktician/tak"
)

type moveGenerator struct {
	ai    *MinimaxAI
	ply   int
	depth int
	p     *tak.Position

	te *tableEntry
	pv []tak.Move
	r  tak.Move

	ms []tak.Move
	i  int
}

type sortMoves struct {
	ms []tak.Move
	vs []int
}

func (s sortMoves) Len() int { return len(s.ms) }
func (s sortMoves) Less(i, j int) bool {
	return s.vs[i] > s.vs[j]
}
func (s sortMoves) Swap(i, j int) {
	s.ms[i], s.ms[j] = s.ms[j], s.ms[i]
	s.vs[i], s.vs[j] = s.vs[j], s.vs[i]
}

func (mg *moveGenerator) sortMoves() {
	s := sortMoves{
		mg.ms,
		mg.ai.stack[mg.ply].vals[:len(mg.ms)],
	}
	for i, m := range s.ms {
		s.vs[i] = mg.ai.history[m.Hash()]
	}
	sort.Sort(s)
}

func (mg *moveGenerator) Reset() {
	mg.i = 0
}

func (mg *moveGenerator) Next() (m tak.Move, p *tak.Position) {
	for {
		var m tak.Move
		switch mg.i {
		case 0:
			mg.i++
			if mg.te != nil {
				m = mg.te.m
				break
			}
			fallthrough
		case 1:
			mg.i++
			if len(mg.pv) > 0 {
				m = mg.pv[0]
				if mg.te != nil && m.Equal(&mg.te.m) {
					continue
				}
				break
			}
			fallthrough
		case 2:
			mg.i++
			if mg.ply == 0 {
				continue
			}
			var ok bool
			if mg.r, ok = mg.ai.response[mg.ai.stack[mg.ply-1].m.Hash()]; ok {
				m = mg.r
				break
			}
			fallthrough
		case 3:
			mg.i++
			if mg.ms == nil {
				mg.ms = mg.p.AllMoves(mg.ai.stack[mg.ply].moves[:0])
			}
			if mg.ply == 0 {
				for i := len(mg.ms) - 1; i > 0; i-- {
					j := mg.ai.rand.Int31n(int32(i))
					mg.ms[j], mg.ms[i] = mg.ms[i], mg.ms[j]
				}
			} else if mg.depth > 1 && !mg.ai.cfg.NoSort {
				mg.sortMoves()
			}
			fallthrough
		default:
			j := mg.i - 4
			mg.i++
			if j >= len(mg.ms) {
				return tak.Move{}, nil
			}
			m = mg.ms[j]
			if mg.te != nil && mg.te.m.Equal(&m) {
				continue
			}
			if len(mg.pv) != 0 && mg.pv[0].Equal(&m) {
				continue
			}
			if mg.r.Equal(&m) {
				continue
			}
		}
		child, e := mg.p.MovePreallocated(&m, mg.ai.stack[mg.ply].p)
		if e == nil {
			return m, child
		}
	}
}
