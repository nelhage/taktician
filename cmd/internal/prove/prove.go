package prove

import (
	"context"
	"flag"
	"log"

	"github.com/google/subcommands"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

type Command struct {
}

func (*Command) Name() string     { return "prove" }
func (*Command) Synopsis() string { return "Prove a position using PN search" }
func (*Command) Usage() string {
	return `prove [options] FILE.ptn`
}

func (*Command) SetFlags(flags *flag.FlagSet) {
}

func (c *Command) Execute(ctx context.Context, flag *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	parsed, e := ptn.ParseFile(flag.Arg(0))
	if e != nil {
		log.Fatal("parse:", e)
	}

	pos, e := parsed.PositionAtMove(0, tak.NoColor)
	if e != nil {
		log.Fatal("get position:", e)
	}

	prove(pos)

	return subcommands.ExitSuccess
}
