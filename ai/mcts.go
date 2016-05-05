package ai

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/nelhage/taktician/cli"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

type MonteCarloAI struct {
	limit time.Duration
	c     float64
	r     *rand.Rand

	Debug int
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
	if ai.limit < limit {
		limit = ai.limit
	}
	for time.Now().Sub(start) < limit {
		node := ai.descend(tree)
		if ai.Debug > 3 {
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
	for _, c := range tree.children {
		if ai.Debug > 2 {
			log.Printf("[mcts][%s]: n=%d w=%d", ptn.FormatMove(&c.move), c.simulations, c.wins)
		}
		if c.simulations > best.simulations {
			best = c
		}
	}
	if ai.Debug > 1 {
		fmt.Printf("[mcts] evaluated simulations=%d wins=%d", tree.simulations, tree.wins)
	}
	return best.move
}

func (ai *MonteCarloAI) populate(t *tree) {
	moves := t.position.AllMoves()
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
	for _, c := range t.children {
		var s float64
		if c.simulations == 0 {
			s = 10
		} else {
			s = float64(c.wins)/float64(c.simulations) +
				ai.c*math.Sqrt(math.Log(float64(t.simulations))/float64(c.simulations))
		}
		if s > val {
			best = c
			val = s
		}
	}
	return ai.descend(best)
}

const maxMoves = 300

func (ai *MonteCarloAI) evaluate(t *tree) bool {
	p := t.position
	for i := 0; i < maxMoves; i++ {
		moves := p.AllMoves()
		var next *tak.Position
		for {
			r := ai.r.Int31n(int32(len(moves)))
			m := moves[r]
			var e error
			if next, e = p.Move(&m); e == nil {
				break
			}
			moves[0], moves[r] = moves[r], moves[0]
			moves = moves[1:]
		}
		if next == nil {
			if ai.Debug > 3 {
				log.Printf("[mcts][aborted due looping]")
				cli.RenderBoard(os.Stderr, p)
			}
			return false
		}
		p = next
		if ok, c := p.GameOver(); ok {
			return c == t.position.ToMove()
		}
	}
	return false
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

func NewMonteCarlo(limit time.Duration) *MonteCarloAI {
	return &MonteCarloAI{
		limit: limit,
		c:     0.7,
		r:     rand.New(rand.NewSource(0)),
	}
}
