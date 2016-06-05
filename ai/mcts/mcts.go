package mcts

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

type MCTSConfig struct {
	Debug int
	Limit time.Duration
	C     float64
	Seed  int64

	Policy func(r *rand.Rand, p *tak.Position, next *tak.Position) *tak.Position
}

type MonteCarloAI struct {
	cfg MCTSConfig

	r *rand.Rand
}

type tree struct {
	position    *tak.Position
	move        tak.Move
	simulations int
	wins        int

	parent   *tree
	children []*tree
}

func (ai *MonteCarloAI) GetMove(p *tak.Position, limit time.Duration) tak.Move {
	tree := &tree{
		position: p,
	}
	ai.populate(tree)
	start := time.Now()
	if ai.cfg.Limit < limit {
		limit = ai.cfg.Limit
	}
	for time.Now().Sub(start) < limit {
		node := ai.descend(tree)
		if ai.cfg.Debug > 4 {
			var s []string
			t := node
			for t.parent != nil {
				s = append(s, ptn.FormatMove(&t.move))
				t = t.parent
			}
			log.Printf("evaluate: [%s]", strings.Join(s, "<-"))
		}
		win := ai.evaluate(node)
		ai.update(node, win)
		ai.populate(node)
	}
	best := tree.children[0]
	i := 0
	for _, c := range tree.children {
		if ai.cfg.Debug > 3 {
			log.Printf("[mcts][%s]: n=%d w=%d", ptn.FormatMove(&c.move), c.simulations, c.wins)
		}
		if c.simulations > best.simulations {
			best = c
			i = 1
		} else if c.simulations == best.simulations {
			i++
			if ai.r.Intn(i) == 0 {
				best = c
				i = 1
			}
		}
	}
	if ai.cfg.Debug > 1 {
		fmt.Printf("[mcts] evaluated simulations=%d wins=%d", tree.simulations, tree.wins)
	}
	return best.move
}

func (ai *MonteCarloAI) populate(t *tree) {
	moves := t.position.AllMoves(nil)
	t.children = make([]*tree, 0, len(moves))
	for _, m := range moves {
		child, e := t.position.Move(&m)
		if e != nil {
			continue
		}
		t.children = append(t.children, &tree{
			position: child,
			move:     m,
			parent:   t,
		})
	}
}

func (ai *MonteCarloAI) descend(t *tree) *tree {
	if t.children == nil {
		return t
	}
	var best *tree
	var val float64
	i := 0
	for _, c := range t.children {
		var s float64
		if c.simulations == 0 {
			s = 10
		} else {
			s = float64(c.wins)/float64(c.simulations) +
				ai.cfg.C*math.Sqrt(math.Log(float64(t.simulations))/float64(c.simulations))
		}
		if s > val {
			best = c
			val = s
			i = 1
		} else if s == val {
			i++
			if ai.r.Intn(i) == 0 {
				best = c
				val = s
				i = 1
			}
		}
	}
	return ai.descend(best)
}

const maxMoves = 300

func (ai *MonteCarloAI) evaluate(t *tree) bool {
	p := t.position
	alloc := tak.Alloc(p.Size())

	for i := 0; i < maxMoves; i++ {
		next := ai.cfg.Policy(ai.r, p, alloc)
		if next == nil {
			return false
		}
		p, alloc = next, p
		if ok, c := p.GameOver(); ok {
			return c == t.position.ToMove()
		}
	}
	return false
}

func RandomPolicy(r *rand.Rand, p *tak.Position, alloc *tak.Position) *tak.Position {
	moves := p.AllMoves(nil)
	var next *tak.Position
	for {
		r := r.Int31n(int32(len(moves)))
		m := moves[r]
		var e error
		if next, e = p.MovePreallocated(&m, alloc); e == nil {
			break
		}
		moves[0], moves[r] = moves[r], moves[0]
		moves = moves[1:]
	}
	return next
}

func (ai *MonteCarloAI) update(t *tree, win bool) {
	for t != nil {
		t.simulations++
		if win {
			t.wins++
		}
		t = t.parent
	}
}

func NewMonteCarlo(cfg MCTSConfig) *MonteCarloAI {
	ai := &MonteCarloAI{
		cfg: cfg,
	}
	if ai.cfg.C == 0 {
		ai.cfg.C = 0.7
	}
	if ai.cfg.Seed == 0 {
		ai.cfg.Seed = time.Now().Unix()
	}
	if ai.cfg.Policy == nil {
		ai.cfg.Policy = RandomPolicy
	}
	ai.r = rand.New(rand.NewSource(ai.cfg.Seed))
	return ai
}
