package mcts

import (
	"fmt"
	"io"
	"log"
	"math"
	"os"

	"github.com/nelhage/taktician/ptn"
)

func (mc *MonteCarloAI) dumpTree(t *tree) {
	f, e := os.Create(mc.cfg.DumpTree)
	if e != nil {
		log.Printf("DumpTree(%s): %v", mc.cfg.DumpTree, e)
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "digraph G {\n")
	mc.dumpTreeNode(f, t)
	fmt.Fprintf(f, "}\n")
}

func (mc *MonteCarloAI) dumpTreeNode(f io.Writer, t *tree) {
	parent := 1
	if t.parent != nil {
		parent = t.parent.simulations
		if t.parent.proven != 0 && t.proven == 0 {
			return
		}

	}
	label := fmt.Sprintf("n=%d p=%d v=%.0f+%.0f",
		t.simulations,
		t.proven,
		float64(t.value)/float64(t.simulations),
		mc.cfg.C*math.Sqrt(math.Log(float64(t.simulations))/float64(parent)))

	fmt.Fprintf(f, `  n%p [label="%s"]`, t, label)
	fmt.Fprintln(f)
	if t.children == nil {
		return
	}

	for _, c := range t.children {
		if t.proven > 0 && c.proven >= 0 {
			continue
		}
		fmt.Fprintf(f, `  n%p -> n%p [label="%s"]`,
			t, c, ptn.FormatMove(c.move))
		fmt.Fprintln(f)
		mc.dumpTreeNode(f, c)
		if c.proven < 0 {
			break
		}
	}
}
