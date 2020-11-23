package tei

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/google/subcommands"
	"github.com/nelhage/taktician/cmd/internal/opt"
	"github.com/nelhage/taktician/tei"
)

type Command struct {
	opt opt.Minimax
}

func (*Command) Name() string     { return "tei" }
func (*Command) Synopsis() string { return "Launch Taktician in TEI mode" }
func (*Command) Usage() string {
	return `tei

Launch the engine in TEI mode, a a UCI-like protocol suitable for being
driven by an external GUI or controller.

`
}

func (c *Command) SetFlags(fs *flag.FlagSet) {
	c.opt.AddFlags(fs)
}

func (c *Command) Execute(ctx context.Context, flag *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	engine := tei.NewEngine(os.Stdin, os.Stdout)
	engine.ConfigFactory = c.opt.BuildConfig
	if err := engine.Run(ctx); err != nil {
		log.Println("tei: ", err.Error())
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
