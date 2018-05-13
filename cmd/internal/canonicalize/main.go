package canonicalize

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strconv"

	"github.com/google/subcommands"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/symmetry"
	"github.com/nelhage/taktician/tak"
)

type Command struct{}

func (*Command) Name() string     { return "canonicalize" }
func (*Command) Synopsis() string { return "Canonicalize the symmetry of a PTN" }
func (*Command) Usage() string {
	return `canonicalize FILE.ptn

Rewrite a PTN into a symmetric PTN in a canonical orientation.
`
}

func (c *Command) SetFlags(flags *flag.FlagSet) {
}

func (c *Command) Execute(ctx context.Context, flag *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if len(flag.Args()) == 0 {
		flag.Usage()
		return subcommands.ExitUsageError
	}

	g, e := ptn.ParseFile(flag.Arg(0))
	if e != nil {
		log.Fatalf("read %s: %v", flag.Arg(0), e)
	}

	var ms []tak.Move
	for _, o := range g.Ops {
		if m, ok := o.(*ptn.Move); ok {
			ms = append(ms, m.Move)
		}
	}

	sz, e := strconv.ParseUint(g.FindTag("Size"), 10, 32)
	if e != nil {
		log.Fatalf("bad size: %v", e)
	}
	out, e := symmetry.Canonical(int(sz), ms)
	if e != nil {
		log.Fatalf("canonicalize: %v", e)
	}

	i := 0
	for _, o := range g.Ops {
		if m, ok := o.(*ptn.Move); ok {
			m.Move = out[i]
			i++
		}
	}

	fmt.Printf(g.Render())
	return subcommands.ExitSuccess
}
