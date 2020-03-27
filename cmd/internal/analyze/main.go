package analyze

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"strings"
	"time"

	"context"

	"github.com/google/subcommands"
	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/ai/mcts"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

type Command struct {
	/* Global options / output options */
	tps        bool
	quiet      bool
	monteCarlo bool
	prove      bool
	debug      int
	cpuProfile string
	memProfile string

	/* Options to select which position(s) to analyze */
	move      int
	all       bool
	black     bool
	white     bool
	variation string

	/* Options which apply to all engines  */
	timeLimit time.Duration
	seed      int64

	/* minimax options */
	eval         bool
	explain      bool
	depth        int
	sort         bool
	tableMem     int64
	nullMove     bool
	extendForces bool
	reduceSlides bool
	multiCut     bool
	precise      bool
	weights      string
	logCuts      string
	symmetry     bool

	/* MCTS options */
	dumpTree string

	/* PN options */
	maxNodes uint64
	maxDepth int
	pn2      bool
}

func (*Command) Name() string     { return "analyze" }
func (*Command) Synopsis() string { return "Evaluate a position from a PTN file" }
func (*Command) Usage() string {
	return `analyze [options] FILE.ptn

Evaluate a position from a PTN file using a configurable engine.

By default evaluates the final position in the file; Use -move and -white/-black
to select a different position, and -variation to play additional moves prior
to analysis.
`
}

func (c *Command) SetFlags(flags *flag.FlagSet) {
	flags.BoolVar(&c.tps, "tps", false, "render position in tps")
	flags.BoolVar(&c.quiet, "quiet", false, "don't print board diagrams")
	flags.BoolVar(&c.monteCarlo, "mcts", false, "Use the MCTS evaluator")
	flags.BoolVar(&c.prove, "prove", false, "Use the PN prover")
	flags.IntVar(&c.debug, "debug", 1, "debug level")
	flags.StringVar(&c.cpuProfile, "cpuprofile", "", "write CPU profile")
	flags.StringVar(&c.memProfile, "memprofile", "", "write memory profile")

	flags.IntVar(&c.move, "move", 0, "PTN move number to analyze")
	flags.BoolVar(&c.all, "all", false, "show all possible moves")
	flags.BoolVar(&c.black, "black", false, "only analyze black's move")
	flags.BoolVar(&c.white, "white", false, "only analyze white's move")
	flags.StringVar(&c.variation, "variation", "", "apply the listed moves after the given position")

	flags.DurationVar(&c.timeLimit, "limit", time.Minute, "limit of how much time to use")
	flags.Int64Var(&c.seed, "seed", 0, "specify a seed")

	flags.BoolVar(&c.eval, "evaluate", false, "only show static evaluation")
	flags.BoolVar(&c.explain, "explain", false, "explain scoring")
	flags.IntVar(&c.depth, "depth", 0, "minimax depth")
	flags.BoolVar(&c.sort, "sort", true, "sort moves via history heuristic")
	flags.Int64Var(&c.tableMem, "table-mem", 0, "set table size")
	flags.BoolVar(&c.nullMove, "null-move", true, "use null-move pruning")
	flags.BoolVar(&c.extendForces, "extend-forces", true, "extend forced moves")
	flags.BoolVar(&c.reduceSlides, "reduce-slides", true, "reduce trivial slides")
	flags.BoolVar(&c.multiCut, "multi-cut", false, "use multi-cut pruning")
	flags.BoolVar(&c.precise, "precise", false, "Limit to optimizations that provably preserve the game-theoretic value")
	flags.StringVar(&c.weights, "weights", "", "JSON-encoded evaluation weights")
	flags.StringVar(&c.logCuts, "log-cuts", "", "log all cuts")
	flags.BoolVar(&c.symmetry, "symmetry", false, "ignore symmetries")

	flags.StringVar(&c.dumpTree, "dump-tree", "", "dump search tree to PATH (MCTS and PN only)")

	flags.Uint64Var(&c.maxNodes, "max-nodes", 0, "Maximum number of nodes to populate in the PN tree")
	flags.IntVar(&c.maxDepth, "max-depth", 0, "Maximum depth to consider in PN search")
	flags.BoolVar(&c.pn2, "pn2", false, "Use PN² search")
}

func (c *Command) Execute(ctx context.Context, flag *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	parsed, e := ptn.ParseFile(flag.Arg(0))
	if e != nil {
		log.Fatal("parse:", e)
	}
	color := tak.NoColor
	switch {
	case c.white && c.black:
		log.Fatal("-white and -black are exclusive")
	case c.white:
		color = tak.White
	case c.black:
		color = tak.Black
	case c.move != 0:
		color = tak.White
	}

	if c.cpuProfile != "" {
		f, e := os.OpenFile(c.cpuProfile, os.O_WRONLY|os.O_CREATE, 0644)
		if e != nil {
			log.Fatalf("open cpu-profile: %s: %v", c.cpuProfile, e)
		}
		pprof.StartCPUProfile(f)
		defer f.Close()
		defer pprof.StopCPUProfile()
	}
	if c.memProfile != "" {
		f, e := os.OpenFile(c.memProfile, os.O_WRONLY|os.O_CREATE, 0644)
		if e != nil {
			log.Fatalf("open memory profile: %s: %v", c.cpuProfile, e)
		}
		defer func() {
			pprof.Lookup("allocs").WriteTo(f, 0)
			f.Close()
		}()
	}

	if !c.all {
		p, e := parsed.PositionAtMove(c.move, color)
		if e != nil {
			log.Fatal("find move:", e)
		}

		if c.variation != "" {
			p, e = applyVariation(p, c.variation)
			if e != nil {
				log.Fatal("-variation:", e)
			}
		}

		c.analyze(p)
	} else {
		p, e := parsed.InitialPosition()
		if e != nil {
			log.Fatal("initial:", e)
		}
		w, b := c.buildAnalysis(p), c.buildAnalysis(p)
		it := parsed.Iterator()
		for it.Next() {
			p := it.Position()
			m := it.PeekMove()
			switch {
			case p.ToMove() == tak.White && color != tak.Black:
				fmt.Printf("%d. %s\n", p.MoveNumber()/2+1, ptn.FormatMove(m))
				c.analyzeWith(w, p)
			case p.ToMove() == tak.Black && color != tak.White:
				fmt.Printf("%d. ... %s\n", p.MoveNumber()/2+1, ptn.FormatMove(m))
				c.analyzeWith(b, p)
			}
		}
		if e := it.Err(); e != nil {
			log.Fatalf("%d: %v", it.PTNMove(), e)
		}
	}
	return subcommands.ExitSuccess
}

func applyVariation(p *tak.Position, variant string) (*tak.Position, error) {
	ms := strings.Split(variant, " ")
	for _, moveStr := range ms {
		m, e := ptn.ParseMove(moveStr)
		if e != nil {
			return nil, e
		}
		p, e = p.Move(m)
		if e != nil {
			return nil, fmt.Errorf("bad move `%s': %v", moveStr, e)
		}
	}
	return p, nil
}

func (c *Command) makeAI(p *tak.Position) *ai.MinimaxAI {
	var w ai.Weights
	if c.weights == "" {
		w = ai.DefaultWeights[p.Size()]
	} else {
		e := json.Unmarshal([]byte(c.weights), &w)
		if e != nil {
			log.Fatalf("parse weights: %v", e)
		}
	}
	cfg := ai.MinimaxConfig{
		Size:  p.Size(),
		Depth: c.depth,
		Seed:  c.seed,
		Debug: c.debug,

		NoSort:         !c.sort,
		TableMem:       c.tableMem,
		NoNullMove:     !c.nullMove,
		NoExtendForces: !c.extendForces,
		NoReduceSlides: !c.reduceSlides,
		MultiCut:       c.multiCut,

		CutLog:        c.logCuts,
		DedupSymmetry: c.symmetry,

		Evaluate: ai.MakeEvaluator(p.Size(), &w),
	}
	if c.precise {
		cfg.MakePrecise()
	}
	return ai.NewMinimax(cfg)
}

func (c *Command) buildAnalysis(p *tak.Position) Analyzer {
	if c.monteCarlo && c.prove {
		log.Fatal("-mcts and -prove are incompatible!")
	}
	if c.prove {
		return &pnAnalysis{
			cmd: c,
		}
	}
	if c.monteCarlo {
		return &monteCarloAnalysis{
			cmd: c,
			ai: mcts.NewMonteCarlo(mcts.MCTSConfig{
				Seed:     c.seed,
				Debug:    c.debug,
				Size:     p.Size(),
				Limit:    c.timeLimit,
				DumpTree: c.dumpTree,
			}),
		}
	}
	return &minimaxAnalysis{cmd: c, ai: c.makeAI(p)}
}

func (c *Command) analyze(p *tak.Position) {
	c.analyzeWith(c.buildAnalysis(p), p)
}

func (c *Command) analyzeWith(analysis Analyzer, p *tak.Position) {
	ctx := context.Background()
	if c.timeLimit != 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, c.timeLimit)
		defer cancel()
	}
	analysis.Analyze(ctx, p)
}
