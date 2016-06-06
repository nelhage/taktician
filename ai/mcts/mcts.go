package mcts

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

type MCTSConfig struct {
	Debug int
	Limit time.Duration
	C     float64
	Seed  int64

	Size int

	Policy PolicyFunc
}

type PolicyFunc func(ctx context.Context, p *tak.Position, next *tak.Position) *tak.Position

type MonteCarloAI struct {
	cfg  MCTSConfig
	mm   *ai.MinimaxAI
	eval ai.EvaluationFunc

	r *rand.Rand
}

type tree struct {
	position    *tak.Position
	move        tak.Move
	simulations int
	wins        int

	value int64

	parent   *tree
	children []*tree
}

func (ai *MonteCarloAI) GetMove(ctx context.Context, p *tak.Position) tak.Move {
	tree := &tree{
		position: p,
	}
	ai.populate(ctx, tree)
	start := time.Now()
	deadline, limited := ctx.Deadline()
	if !limited || deadline.Sub(start) > ai.cfg.Limit {
		deadline = time.Now().Add(ai.cfg.Limit)
	}
	ctx = WithRand(ctx, ai.r)

	next := start.Add(10 * time.Second)
	for time.Now().Before(deadline) {
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
		ai.populate(ctx, node)
		var win bool
		if node.value == 0 {
			win = ai.evaluate(ctx, node)
		}
		ai.update(node, win)
		if time.Now().After(next) && ai.cfg.Debug > 0 {
			ai.printpv(tree)
			next = time.Now().Add(10 * time.Second)
		}
	}
	best := tree.children[0]
	i := 0
	for _, c := range tree.children {
		if ai.cfg.Debug > 2 {
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

func (mc *MonteCarloAI) printpv(t *tree) {
	depth := 0
	ts := []*tree{t}
	for t.children != nil && t.simulations > visitThreshold {
		best := t.children[0]
		for _, c := range t.children {
			if c.simulations > best.simulations {
				best = c
			}
		}
		t = best
		ts = append(ts, best)
		depth++
	}
	ms := make([]tak.Move, depth)
	for t.parent != nil {
		ms[depth-1] = t.move
		t = t.parent
		depth--
	}
	var ptns []string
	for _, m := range ms {
		ptns = append(ptns, ptn.FormatMove(&m))
	}
	log.Printf("pv=[%s] n=%d w=%d",
		strings.Join(ptns, " "),
		ts[1].simulations, ts[1].wins,
	)
}

func (mc *MonteCarloAI) populate(ctx context.Context, t *tree) {
	_, v, _ := mc.mm.Analyze(ctx, t.position)
	if v < -ai.WinThreshold || v > ai.WinThreshold {
		t.value = v
		return
	}

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

const visitThreshold = 10

func (mc *MonteCarloAI) descendPolicy(t *tree) *tree {
	var best *tree
	val := ai.MinEval
	i := 0
	for _, c := range t.children {
		v := mc.eval(mc.mm, c.position)
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
	if t.simulations < visitThreshold {
		return ai.descendPolicy(t)
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
			}
		}
	}
	return ai.descend(best)
}

const maxMoves = 50
const evalThreshold = 500

func (ai *MonteCarloAI) evaluate(ctx context.Context, t *tree) bool {
	p := t.position
	alloc := tak.Alloc(p.Size())

	for i := 0; i < maxMoves; i++ {
		if ok, c := p.GameOver(); ok {
			return c == t.position.ToMove()
		}
		next := ai.cfg.Policy(ctx, p, alloc)
		if next == nil {
			return false
		}
		p, alloc = next, p
	}
	v := ai.eval(ai.mm, p)
	return v > evalThreshold
}

func (mc *MonteCarloAI) update(t *tree, win bool) {
	for t != nil {
		foundWin := false
		foundLose := true
		for _, c := range t.children {
			if c.value < 0 {
				foundWin = true
				break
			}
			if c.value == 0 {
				foundLose = false
			}
		}
		if foundWin {
			t.value = 1
		} else if foundLose {
			t.value = -1
		}

		t.simulations++
		if win {
			t.wins++
		}
		t = t.parent
	}
}

func NewMonteCarlo(cfg MCTSConfig) *MonteCarloAI {
	mc := &MonteCarloAI{
		cfg: cfg,
	}
	if mc.cfg.C == 0 {
		mc.cfg.C = 0.7
	}
	if mc.cfg.Seed == 0 {
		mc.cfg.Seed = time.Now().Unix()
	}
	if mc.cfg.Policy == nil {
		mc.cfg.Policy = NewMinimaxPolicy(&mc.cfg, 1)
	}
	mc.r = rand.New(rand.NewSource(mc.cfg.Seed))
	mc.mm = ai.NewMinimax(ai.MinimaxConfig{
		Size:     cfg.Size,
		Evaluate: ai.EvaluateWinner,
		NoTable:  true,
		Depth:    1,
		Seed:     mc.cfg.Seed,
	})
	mc.eval = ai.MakeEvaluator(mc.cfg.Size, nil)
	return mc
}
