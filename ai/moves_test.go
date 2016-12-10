package ai

import (
	"math/rand"
	"testing"

	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
	"github.com/nelhage/taktician/taktest"
)

func TestMoveGenerator(t *testing.T) {
	p, _ := ptn.ParseTPS("1,1,x3/x,1,x,2,x/x,2,1C,x2/x,2,1,x2/2,2,1,x2 2 6")
	pvm := taktest.Move("Cc4")
	tem := taktest.Move("c4")
	cm := taktest.Move("b1>")
	te := tableEntry{
		m: tem,
	}

	ai := NewMinimax(MinimaxConfig{Size: 5})
	ai.rand = rand.New(rand.NewSource(7))
	ai.history[cm] = 100

	mg := &ai.stack[1].mg
	*mg = moveGenerator{
		p:     p,
		f:     &ai.stack[1],
		ai:    ai,
		ply:   1,
		depth: 5,

		te: &te,
		pv: []tak.Move{pvm},
	}

	allS := make(map[string]struct{})
	all := p.AllMoves(nil)
	for _, a := range all {
		if _, e := p.Move(a); e == nil {
			allS[ptn.FormatMove(a)] = struct{}{}
		}
	}

	var generated []tak.Move
	genS := make(map[string]struct{})
	for {
		m, c := mg.Next()
		if c == nil {
			break
		}
		generated = append(generated, m)
		genS[ptn.FormatMove(m)] = struct{}{}
	}

	if g := generated[0]; !g.Equal(tem) {
		t.Errorf("move[0]=%s != %s",
			ptn.FormatMove(g), ptn.FormatMove(tem))
	}
	if g := generated[1]; !g.Equal(pvm) {
		t.Errorf("move[1]=%s != %s",
			ptn.FormatMove(g), ptn.FormatMove(pvm))
	}
	if g := generated[2]; !g.Equal(cm) {
		t.Errorf("move[2]=%s != %s",
			ptn.FormatMove(g), ptn.FormatMove(cm))
	}

	for g := range genS {
		if _, ok := allS[g]; !ok {
			t.Errorf("generated additional move %s", g)
		}
	}
	for a := range allS {
		if _, ok := genS[a]; !ok {
			t.Errorf("generate missed move %s", a)
		}
	}

}
