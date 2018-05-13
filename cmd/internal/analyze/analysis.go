package analyze

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/ai/mcts"
	"github.com/nelhage/taktician/cli"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

type Analyzer interface {
	Analyze(ctx context.Context, p *tak.Position)
}

type minimaxAnalysis struct {
	ai *ai.MinimaxAI
}

func (m *minimaxAnalysis) Analyze(ctx context.Context, p *tak.Position) {
	if !*quiet {
		cli.RenderBoard(nil, os.Stdout, p)
		if *explain {
			ai.ExplainScore(m.ai, os.Stdout, p)
		}
	}
	if *eval {
		val := m.ai.Evaluate(p)
		if p.ToMove() == tak.Black {
			val = -val
		}
		fmt.Printf(" Val=%d\n", val)
		return
	}
	pvs, val, _ := m.ai.AnalyzeAll(ctx, p)
	fmt.Printf("AI analysis:\n")
	for _, pv := range pvs {
		fmt.Printf(" pv=")
		for _, m := range pv {
			fmt.Printf("%s ", ptn.FormatMove(m))
		}
		fmt.Printf("\n")
	}
	fmt.Printf(" value=%d\n", val)
	if *tps {
		fmt.Printf("[TPS \"%s\"]\n", ptn.FormatTPS(p))
	}
	fmt.Println()

	if len(pvs) == 0 || *quiet {
		return
	}

	for _, m := range pvs[0] {
		n, e := p.Move(m)
		if e != nil {
			log.Printf("illegal move in pv: %s: %v", ptn.FormatMove(m), e)
			if val < ai.WinThreshold && val > -ai.WinThreshold {
				log.Fatal("illegal move in non-terminal pv!")
			}
			return
		}
		p = n
	}

	fmt.Println("Resulting position:")
	cli.RenderBoard(nil, os.Stdout, p)
	if *explain {
		ai.ExplainScore(m.ai, os.Stdout, p)
	}
	fmt.Println()
	fmt.Println()
}

type monteCarloAnalysis struct {
	ai *mcts.MonteCarloAI
}

func (m *monteCarloAnalysis) Analyze(ctx context.Context, p *tak.Position) {
	if !*quiet {
		cli.RenderBoard(nil, os.Stdout, p)
	}
	pv := m.ai.GetMove(ctx, p)
	fmt.Printf("AI analysis:\n")
	fmt.Printf("  PV=%s\n", ptn.FormatMove(pv))
}
