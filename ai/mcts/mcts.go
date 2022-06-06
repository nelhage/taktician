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

	MMDepth       int
	MaxRollout    int
	EvalThreshold int64
	Policy        string
	ForceCorners  bool

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

func (t *tree) ucb(C float64, N int) float64 {
	if t.proven > 0 {
		return -100
	} else if t.proven < 0 {
		return 100
	} else if t.simulations == 0 {
		return 10
	} else {
		return -float64(t.value)/float64(t.simulations) +
			C*math.Sqrt(math.Log(float64(N))/float64(t.simulations))
	}
}

type bySims []*tree

func (b bySims) Len() int           { return len(b) }
func (b bySims) Less(i, j int) bool { return b[j].simulations < b[i].simulations }
func (b bySims) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (ai *MonteCarloAI) cornerMove(p *tak.Position) tak.Move {
	for {
		row := (p.Size() - 1) * ai.r.Intn(2)
		col := (p.Size() - 1) * ai.r.Intn(2)
		if len(p.At(row, col)) > 0 {
			continue
		}
		return tak.Move{
			X:    int8(row),
			Y:    int8(row),
			Type: tak.PlaceFlat,
		}
	}
}

func (ai *MonteCarloAI) GetMove(ctx context.Context, p *tak.Position) tak.Move {
	if ai.cfg.ForceCorners && p.MoveNumber() < 2 {
		return ai.cornerMove(p)
	}

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
			log.Printf("evaluate: [%s] = %d p=%d",
				strings.Join(s, "<-"), val, node.proven)
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

	if ai.cfg.DumpTree != "" {
		ai.dumpTree(tree)
	}

	best := tree.children[0]
	i := 0
	sort.Sort(bySims(tree.children))
	if ai.cfg.Debug > 1 {
		log.Printf("=== mcts done ===")
	}
	for _, c := range tree.children {
		if ai.cfg.Debug > 1 {
			log.Printf("[mcts][%s]: n=%d v=%d:%d(%0.3f) ucb=%f",
				ptn.FormatMove(c.move), c.simulations, c.proven, c.value,
				float64(c.value)/float64(c.simulations),
				c.ucb(ai.cfg.C, tree.simulations),
			)
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
	if tree.proven != 0 {
		if len(tree.children) == 0 {
			return ai.mm.GetMove(ctx, p)
		}
		best := tree.children[0]
		for _, c := range tree.children {
			if c.proven < best.proven {
				best = c
			}
		}
		if ai.cfg.Debug > 1 {
			log.Printf("proven m=%s v=%d", ptn.FormatMove(best.move), -best.proven)
		}
		return best.move
	}
	if ai.cfg.Debug > 0 {
		log.Printf("[mcts] evaluated simulations=%d value=%d proven=%d", tree.simulations, tree.value, tree.proven)
	}
	return best.move
}

func (mc *MonteCarloAI) printdbg(t *tree) {
	log.Printf("===")
	for _, c := range t.children {
		if c.simulations*20 > t.simulations {
			log.Printf("[mcts][%s]: n=%d v=%d:%d(%0.3f) ucb=%f",
				ptn.FormatMove(c.move), c.simulations, c.proven, c.value,
				float64(c.value)/float64(c.simulations),
				c.ucb(mc.cfg.C, t.simulations),
			)
		}
	}
}

func (mc *MonteCarloAI) populate(ctx context.Context, t *tree) {
	/*
		_, v, _ := mc.mm.Analyze(ctx, t.position)
		if v > ai.WinThreshold {
			t.proven = 1
			return
		} else if v < -ai.WinThreshold {
			t.proven = -1
			return
		}
	*/

	moves := t.position.AllMoves(nil)
	t.children = make([]*tree, 0, len(moves))
	for _, m := range moves {
		child, e := t.position.Move(m)
		if e != nil {
			continue
		}
		proven := 0
		if ok, winner := child.GameOver(); ok && winner != tak.NoColor {
			if winner == child.ToMove() {
				proven = 1
			} else {
				proven = -1
			}
		}
		t.children = append(t.children, &tree{
			position: child,
			move:     m,
			parent:   t,
			proven:   proven,
		})
	}
}

func (ai *MonteCarloAI) descend(t *tree) *tree {
	for {
		if len(t.children) == 0 {
			return t
		}
		var best *tree
		var val float64 = math.Inf(-1)
		i := 0
		for _, c := range t.children {
			s := c.ucb(ai.cfg.C, t.simulations)

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
			best = t.children[0]
		}
		t = best
	}
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
