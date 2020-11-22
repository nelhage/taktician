package tei

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/google/subcommands"
	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/tak"
	"github.com/nelhage/taktician/tei"
)

type Command struct {
	mm  *ai.MinimaxAI
	pos *tak.Position
}

func (*Command) Name() string     { return "engine" }
func (*Command) Synopsis() string { return "Launch Taktician in UCI-like engine mode" }
func (*Command) Usage() string {
	return `engine
Launch the engine in a UCI-like mode, suitable for being
driven by an external GUI or controller.`
}

func (c *Command) SetFlags(fs *flag.FlagSet) {
}

func (c *Command) Execute(ctx context.Context, flag *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	engine := tei.NewEngine(os.Stdin, os.Stdout)
	if err := engine.Run(ctx); err != nil {
		log.Println("tei: ", err.Error())
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
