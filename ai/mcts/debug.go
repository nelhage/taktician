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
	}
	label := fmt.Sprintf("n=%d v=%.0f+%.0f",
		t.simulations,
		float64(t.value)/float64(t.simulations),
		mc.cfg.C*math.Sqrt(math.Log(float64(t.simulations))/float64(parent)))

	fmt.Fprintf(f, `  n%p [label="%s"]`, t, label)
	fmt.Fprintln(f)
	if t.children == nil || t.simulations < mc.cfg.InitialVisits {
		return
	}

	for _, c := range t.children {
		if c.simulations < mc.cfg.InitialVisits {
			continue
		}
		fmt.Fprintf(f, `  n%p -> n%p [label="%s"]`,
			t, c, ptn.FormatMove(c.move))
		fmt.Fprintln(f)
		mc.dumpTreeNode(f, c)
	}
}
