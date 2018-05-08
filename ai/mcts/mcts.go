package mcts

import (
	"log"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"context"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/bitboard"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

type MCTSConfig struct {
	Debug int
	Limit time.Duration
	C     float64
	Seed  int64

	InitialVisits int
	MMDepth       int
	MaxRollout    int
	EvalThreshold int64
	Policy        string

	Size int

	DumpTree string
}

type Policy interface {
	Select(ctx context.Context, m *MonteCarloAI, p *tak.Position) *tak.Position
}

type MonteCarloAI struct {
	c bitboard.Constants

	cfg  MCTSConfig
	mm   *ai.MinimaxAI
	eval ai.EvaluationFunc

	policy Policy

	r *rand.Rand
}

type tree struct {
	position    *tak.Position
	move        tak.Move
	simulations int

	proven int
	value  int

	parent   *tree
	children []*tree
}

type bySims []*tree

func (b bySims) Len() int           { return len(b) }
func (b bySims) Less(i, j int) bool { return b[j].simulations < b[i].simulations }
func (b bySims) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (ai *MonteCarloAI) GetMove(ctx context.Context, p *tak.Position) tak.Move {
	tree := &tree{
		position: p,
	}
	start := time.Now()
	deadline, limited := ctx.Deadline()
	if !limited || deadline.Sub(start) > ai.cfg.Limit {
		deadline = time.Now().Add(ai.cfg.Limit)
	}

	var tick <-chan time.Time
	if ai.cfg.Debug > 2 {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		tick = ticker.C
	}
	for time.Now().Before(deadline) {
		node := ai.descend(tree)
		ai.populate(ctx, node)
		if tree.proven != 0 {
			break
		}
		var val int
		if node.proven == 0 {
			val = ai.rollout(ctx, node)
		}
		if ai.cfg.Debug > 4 {
			var s []string
			t := node
			for t.parent != nil {
				s = append(s, ptn.FormatMove(t.move))
				t = t.parent
			}
			log.Printf("evaluate: [%s] = %d",
				strings.Join(s, "<-"), val)
		}
		ai.update(node, val)
		if tick != nil {
			select {
			case <-tick:
				ai.printdbg(tree)
			default:
			}
		}
	}
	if tree.proven != 0 {
		return ai.mm.GetMove(ctx, p)
	}
	best := tree.children[0]
	i := 0
	sort.Sort(bySims(tree.children))
	if ai.cfg.Debug > 2 {
		log.Printf("=== mcts done ===")
	}
	for _, c := range tree.children {
		if ai.cfg.Debug > 2 {
			log.Printf("[mcts][%s]: n=%d v=%d:%d(%0.3f)",
				ptn.FormatMove(c.move), c.simulations, c.proven, c.value,
				float64(c.value)/float64(c.simulations))
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
		log.Printf("[mcts] evaluated simulations=%d value=%d", tree.simulations, tree.value)
	}
	if ai.cfg.DumpTree != "" {
		ai.dumpTree(tree)
	}
	return best.move
}

func (mc *MonteCarloAI) printdbg(t *tree) {
	log.Printf("===")
	for _, c := range t.children {
		if c.simulations*20 > t.simulations {
			log.Printf("[mcts][%s]: n=%d v=%d:%d(%0.3f)",
				ptn.FormatMove(c.move), c.simulations, c.proven, c.value,
				float64(c.value)/float64(c.simulations))
		}
	}
}

func (mc *MonteCarloAI) populate(ctx context.Context, t *tree) {
	_, v, _ := mc.mm.Analyze(ctx, t.position)
	if v > ai.WinThreshold {
		t.proven = 1
		return
	} else if v < -ai.WinThreshold {
		t.proven = -1
		return
	}

	moves := t.position.AllMoves(nil)
	t.children = make([]*tree, 0, len(moves))
	for _, m := range moves {
		child, e := t.position.Move(m)
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

func (mc *MonteCarloAI) descendPolicy(t *tree) *tree {
	var best *tree
	val := ai.MinEval
	i := 0
	for _, c := range t.children {
		v := mc.eval(&mc.c, c.position)
		if v > val {
			best = c
			val = v
			i = 1
		} else if v == val {
			i++
			if mc.r.Intn(i) == 0 {
				best = c
			}
		}
	}
	return best
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
		if c.proven > 0 {
			s = 0.01
		} else if c.proven < 0 {
			s = 100
		} else if c.simulations == 0 {
			s = 10
		} else {
			s = -float64(c.value)/float64(c.simulations) +
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
			}
		}
	}
	if best == nil {
		return t.children[0]
	}
	return ai.descend(best)
}

func (ai *MonteCarloAI) rollout(ctx context.Context, t *tree) int {
	p := t.position.Clone()

	for i := 0; i < ai.cfg.MaxRollout; i++ {
		if ok, c := p.GameOver(); ok {
			switch c {
			case tak.NoColor:
				return 0
			case t.position.ToMove():
				return 1
			default:
				return -1
			}
		}
		next := ai.policy.Select(ctx, ai, p)
		if next == nil {
			return 0
		}
		p = next
	}
	v := ai.eval(&ai.c, p)
	if v > ai.cfg.EvalThreshold {
		return 1
	} else if v < -ai.cfg.EvalThreshold {
		return -1
	}
	return 0
}

func (mc *MonteCarloAI) update(t *tree, value int) {
	for t != nil {
		t.simulations++
		if t.proven != 0 {
			if t.parent == nil {
				return
			}
			// Minimax backup
			if t.proven < 0 {
				// My best move is a loss; therefore
				// my parent should choose this branch
				// and win
				t.parent.proven = 1
				value = -1
			} else {
				// This move is a win for me; My
				// parent is a loss only if *all* of
				// its children are wins
				all := true
				for _, ch := range t.parent.children {
					if ch.proven <= 0 {
						all = false
						break
					}
				}
				if all {
					t.parent.proven = -1
				}
				value = 1
			}
		} else {
			t.value += value
		}
		value = -value
		t = t.parent
	}
}

func NewMonteCarlo(cfg MCTSConfig) *MonteCarloAI {
	mc := &MonteCarloAI{
		cfg: cfg,
		c:   bitboard.Precompute(uint(cfg.Size)),
	}
	if mc.cfg.C == 0 {
		mc.cfg.C = 0.7
	}
	if mc.cfg.Seed == 0 {
		mc.cfg.Seed = time.Now().Unix()
	}
	if mc.cfg.InitialVisits == 0 {
		mc.cfg.InitialVisits = 3
	}
	if mc.cfg.MMDepth == 0 {
		mc.cfg.MMDepth = 3
	}
	if mc.cfg.MaxRollout == 0 {
		mc.cfg.MaxRollout = 50
	}
	if mc.cfg.EvalThreshold == 0 {
		mc.cfg.EvalThreshold = 2000
	}
	mc.policy = mc.buildPolicy()
	mc.r = rand.New(rand.NewSource(mc.cfg.Seed))
	mc.mm = ai.NewMinimax(ai.MinimaxConfig{
		Size:     cfg.Size,
		Evaluate: ai.EvaluateWinner,
		Depth:    mc.cfg.MMDepth,
		Seed:     mc.cfg.Seed,
	})
	mc.eval = ai.MakeEvaluator(mc.cfg.Size, nil)
	return mc
}
