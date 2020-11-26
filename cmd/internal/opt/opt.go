package opt

import (
	"encoding/json"
	"flag"
	"log"

	"github.com/nelhage/taktician/ai"
)

type Minimax struct {
	Seed         int64
	Debug        int
	Depth        int
	MaxEvals     uint64
	Sort         bool
	TableMem     int64
	NullMove     bool
	ExtendForces bool
	ReduceSlides bool
	MultiCut     bool
	Precise      bool
	Weights      string
	LogCuts      string
	Symmetry     bool
}

func (o *Minimax) AddFlags(flags *flag.FlagSet) {
	flags.IntVar(&o.Debug, "debug", 1, "debug level")
	flags.Int64Var(&o.Seed, "seed", 0, "specify a seed")
	flags.IntVar(&o.Depth, "depth", 0, "minimax depth")
	flags.Uint64Var(&o.MaxEvals, "max-evals", 0, "Limit the search by number of nodes evaluated")
	flags.BoolVar(&o.Sort, "sort", true, "sort moves via history heuristic")
	flags.Int64Var(&o.TableMem, "table-mem", 0, "set table size")
	flags.BoolVar(&o.NullMove, "null-move", true, "use null-move pruning")
	flags.BoolVar(&o.ExtendForces, "extend-forces", true, "extend forced moves")
	flags.BoolVar(&o.ReduceSlides, "reduce-slides", true, "reduce trivial slides")
	flags.BoolVar(&o.MultiCut, "multi-cut", false, "use multi-cut pruning")
	flags.BoolVar(&o.Precise, "precise", false, "Limit to optimizations that provably preserve the game-theoretic value")
	flags.StringVar(&o.Weights, "weights", "", "JSON-encoded evaluation weights")
	flags.StringVar(&o.LogCuts, "log-cuts", "", "log all cuts")
	flags.BoolVar(&o.Symmetry, "symmetry", false, "ignore symmetries")
}

func (o *Minimax) BuildConfig(size int) ai.MinimaxConfig {
	var w ai.Weights
	if o.Weights == "" {
		w = ai.DefaultWeights[size]
	} else {
		e := json.Unmarshal([]byte(o.Weights), &w)
		if e != nil {
			log.Fatalf("parse weights: %v", e)
		}
	}
	cfg := ai.MinimaxConfig{
		Size:     size,
		Depth:    o.Depth,
		MaxEvals: o.MaxEvals,
		Seed:     o.Seed,
		Debug:    o.Debug,

		NoSort:         !o.Sort,
		TableMem:       o.TableMem,
		NoNullMove:     !o.NullMove,
		NoExtendForces: !o.ExtendForces,
		NoReduceSlides: !o.ReduceSlides,
		MultiCut:       o.MultiCut,

		CutLog:        o.LogCuts,
		DedupSymmetry: o.Symmetry,

		Evaluate: ai.MakeEvaluator(size, &w),
	}
	if o.Precise {
		cfg.MakePrecise()
	}
	return cfg
}
