package main

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

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/ai/mcts"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

var (
	/* Global options / output options */
	tps        = flag.Bool("tps", false, "render position in tps")
	quiet      = flag.Bool("quiet", false, "don't print board diagrams")
	monteCarlo = flag.Bool("mcts", false, "Use the MCTS evaluator")
	debug      = flag.Int("debug", 1, "debug level")

	/* Options to select which position(s) to analyze */
	move      = flag.Int("move", 0, "PTN move number to analyze")
	all       = flag.Bool("all", false, "show all possible moves")
	black     = flag.Bool("black", false, "only analyze black's move")
	white     = flag.Bool("white", false, "only analyze white's move")
	variation = flag.String("variation", "", "apply the listed moves after the given position")

	/* Options which apply to both engines  */
	timeLimit = flag.Duration("limit", time.Minute, "limit of how much time to use")
	seed      = flag.Int64("seed", 0, "specify a seed")

	/* minimax options */
	eval         = flag.Bool("evaluate", false, "only show static evaluation")
	explain      = flag.Bool("explain", false, "explain scoring")
	depth        = flag.Int("depth", 0, "minimax depth")
	sort         = flag.Bool("sort", true, "sort moves via history heuristic")
	tableMem     = flag.Int64("table-mem", 0, "set table size")
	nullMove     = flag.Bool("null-move", true, "use null-move pruning")
	extendForces = flag.Bool("extend-forces", true, "extend forced moves")
	reduceSlides = flag.Bool("reduce-slides", true, "reduce trivial slides")
	multiCut     = flag.Bool("multi-cut", false, "use multi-cut pruning")
	precise      = flag.Bool("precise", false, "Limit to optimizations that provably preserve the game-theoretic value")
	weights      = flag.String("weights", "", "JSON-encoded evaluation weights")
	logCuts      = flag.String("log-cuts", "", "log all cuts")

	cpuProfile = flag.String("cpuprofile", "", "write CPU profile")
)

func main() {
	flag.Parse()

	parsed, e := ptn.ParseFile(flag.Arg(0))
	if e != nil {
		log.Fatal("parse:", e)
	}
	color := tak.NoColor
	switch {
	case *white && *black:
		log.Fatal("-white and -black are exclusive")
	case *white:
		color = tak.White
	case *black:
		color = tak.Black
	case *move != 0:
		color = tak.White
	}

	if *cpuProfile != "" {
		f, e := os.OpenFile(*cpuProfile, os.O_WRONLY|os.O_CREATE, 0644)
		if e != nil {
			log.Fatalf("open cpu-profile: %s: %v", *cpuProfile, e)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if !*all {
		p, e := parsed.PositionAtMove(*move, color)
		if e != nil {
			log.Fatal("find move:", e)
		}

		if *variation != "" {
			p, e = applyVariation(p, *variation)
			if e != nil {
				log.Fatal("-variation:", e)
			}
		}

		analyze(p)
	} else {
		p, e := parsed.InitialPosition()
		if e != nil {
			log.Fatal("initial:", e)
		}
		w, b := buildAnalysis(p), buildAnalysis(p)
		it := parsed.Iterator()
		for it.Next() {
			p := it.Position()
			m := it.PeekMove()
			switch {
			case p.ToMove() == tak.White && color != tak.Black:
				fmt.Printf("%d. %s\n", p.MoveNumber()/2+1, ptn.FormatMove(m))
				analyzeWith(w, p)
			case p.ToMove() == tak.Black && color != tak.White:
				fmt.Printf("%d. ... %s\n", p.MoveNumber()/2+1, ptn.FormatMove(m))
				analyzeWith(b, p)
			}
		}
		if e := it.Err(); e != nil {
			log.Fatalf("%d: %v", it.PTNMove(), e)
		}
	}
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

func makeAI(p *tak.Position) *ai.MinimaxAI {
	var w ai.Weights
	if *weights == "" {
		w = ai.DefaultWeights[p.Size()]
	} else {
		e := json.Unmarshal([]byte(*weights), &w)
		if e != nil {
			log.Fatalf("parse weights: %v", e)
		}
	}
	cfg := ai.MinimaxConfig{
		Size:  p.Size(),
		Depth: *depth,
		Seed:  *seed,
		Debug: *debug,

		NoSort:         !*sort,
		TableMem:       *tableMem,
		NoNullMove:     !*nullMove,
		NoExtendForces: !*extendForces,
		NoReduceSlides: !*reduceSlides,
		MultiCut:       *multiCut,

		CutLog: *logCuts,

		Evaluate: ai.MakeEvaluator(p.Size(), &w),
	}
	if *precise {
		cfg.MakePrecise()
	}
	return ai.NewMinimax(cfg)
}

func buildAnalysis(p *tak.Position) Analyzer {
	if *monteCarlo {
		return &monteCarloAnalysis{
			mcts.NewMonteCarlo(mcts.MCTSConfig{
				Seed:  *seed,
				Debug: *debug,
				Size:  p.Size(),
				Limit: *timeLimit,
			}),
		}
	}
	return &minimaxAnalysis{makeAI(p)}
}

func analyze(p *tak.Position) {
	analyzeWith(buildAnalysis(p), p)
}

func analyzeWith(analysis Analyzer, p *tak.Position) {
	ctx := context.Background()
	if *timeLimit != 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, *timeLimit)
		defer cancel()
	}
	analysis.Analyze(ctx, p)
}
