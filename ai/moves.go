package ai

import "github.com/nelhage/taktician/tak"

type moveGenerator struct {
	ai  *MinimaxAI
	ply int
	p   *tak.Position

	te *tableEntry
	pv []tak.Move

	ms []tak.Move
	i  int
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
			mg.ms = mg.p.AllMoves()
			if mg.ply == 0 {
				for i := len(mg.ms) - 1; i > 0; i-- {
					j := mg.ai.rand.Int31n(int32(i))
					mg.ms[j], mg.ms[i] = mg.ms[i], mg.ms[j]
				}
			}
			fallthrough
		default:
			mg.i++
			if len(mg.ms) == 0 {
				return tak.Move{}, nil
			}
			m = mg.ms[0]
			mg.ms = mg.ms[1:]
			if mg.te != nil && mg.te.m.Equal(&m) {
				continue
			}
			if len(mg.pv) != 0 && mg.pv[0].Equal(&m) {
				continue
			}
		}
		child, e := mg.p.Move(&m)
		if e == nil {
			return m, child
		}
	}
}
