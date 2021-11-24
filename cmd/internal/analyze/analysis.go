package analyze

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/ai/mcts"
	"github.com/nelhage/taktician/cli"
	"github.com/nelhage/taktician/prove"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

type Analyzer interface {
	Analyze(ctx context.Context, p *tak.Position)
}

type minimaxAnalysis struct {
	cmd *Command
	ai  *ai.MinimaxAI
}

func (m *minimaxAnalysis) Analyze(ctx context.Context, p *tak.Position) {
	if !m.cmd.quiet {
		cli.RenderBoard(nil, os.Stdout, p)
		if m.cmd.explain {
			ai.ExplainScore(m.ai, os.Stdout, p)
		}
	}
	if m.cmd.eval {
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
	if m.cmd.tps {
		fmt.Printf("[TPS \"%s\"]\n", ptn.FormatTPS(p))
	}
	fmt.Println()

	if len(pvs) == 0 || m.cmd.quiet {
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
	if m.cmd.explain {
		ai.ExplainScore(m.ai, os.Stdout, p)
	}
	fmt.Println()
	fmt.Println()
}

type monteCarloAnalysis struct {
	cmd *Command
	ai  *mcts.MonteCarloAI
}

func (m *monteCarloAnalysis) Analyze(ctx context.Context, p *tak.Position) {
	if !m.cmd.quiet {
		cli.RenderBoard(nil, os.Stdout, p)
	}
	pv := m.ai.GetMove(ctx, p)
	fmt.Printf("AI analysis:\n")
	fmt.Printf("  PV=%s\n", ptn.FormatMove(pv))
}

type pnAnalysis struct {
	cmd *Command
}

func (a *pnAnalysis) Analyze(ctx context.Context, p *tak.Position) {
	prover := prove.New(prove.Config{
		Debug:          a.cmd.mmopt.Debug,
		MaxNodes:       a.cmd.maxNodes,
		PreserveSolved: a.cmd.dumpTree != "",
		PN2:            a.cmd.pn2,
		MaxDepth:       a.cmd.maxDepth,
	})

	if !a.cmd.quiet {
		cli.RenderBoard(nil, os.Stdout, p)
	}

	out, stats := prover.Prove(ctx, p)
	var result string
	switch out.Result {
	case prove.EvalTrue:
		result = "WIN"
	case prove.EvalFalse:
		result = "DRAW|LOSE"
	case prove.EvalUnknown:
		result = "UNKNOWN"
	}
	fmt.Printf("PN search analysis:\n")
	var move string
	if out.Move.Type != 0 {
		move = ptn.FormatMove(out.Move)
	} else {
		move = "(none)"
	}
	fmt.Printf(" value=%s move=%s duration=%s searched=%d proof=%d disproof=%d depth=%d maxDepth=%d\n",
		result,
		move,
		out.Duration,
		stats.Nodes,
		out.Proof,
		out.Disproof,
		out.Depth,
		stats.MaxDepth,
	)

	if a.cmd.dumpTree != "" {
		out, e := os.OpenFile(a.cmd.dumpTree, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if e != nil {
			log.Fatalf("dump-tree(%s): %v", a.cmd.dumpTree, e)
		}
		buf := bufio.NewWriter(out)
		prover.DumpTree(buf)
		if e := buf.Flush(); e != nil {
			log.Fatalf("dump-tree(%s): %v", a.cmd.dumpTree, e)
		}
		out.Close()
	}
}

type dfpnAnalysis struct {
	cmd *Command
}

func (a *dfpnAnalysis) Analyze(ctx context.Context, p *tak.Position) {
	prover := prove.NewDFPN(&prove.DFPNConfig{
		Debug:    a.cmd.mmopt.Debug,
		TableMem: a.cmd.mmopt.TableMem,
	})

	if !a.cmd.quiet {
		cli.RenderBoard(nil, os.Stdout, p)
	}

	out, stats := prover.Prove(p)
	var result string
	switch out.Result {
	case prove.EvalTrue:
		result = "WIN"
	case prove.EvalFalse:
		result = "DRAW|LOSE"
	case prove.EvalUnknown:
		result = "UNKNOWN"
	}
	fmt.Printf("PN search analysis:\n")
	var move string
	if out.Move.Type != 0 {
		move = ptn.FormatMove(out.Move)
	} else {
		move = "(none)"
	}
	fmt.Printf(" value=%s move=%s duration=%s\n",
		result,
		move,
		out.Duration,
	)
	fmt.Printf(" work=%d terminal=%d solved=%d repetition=%d hit=%d/%d (%0.2f%%)\n",
		stats.Work,
		stats.Terminal,
		stats.Solved,
		stats.Repetition,
		stats.Hits,
		stats.Hits+stats.Miss,
		100*float64(stats.Hits)/float64(stats.Hits+stats.Miss),
	)
}
