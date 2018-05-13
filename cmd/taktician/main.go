package main

import (
	"context"
	"flag"
	"os"

	"github.com/google/subcommands"
	"github.com/nelhage/taktician/cmd/internal/analyze"
	"github.com/nelhage/taktician/cmd/internal/canonicalize"
	"github.com/nelhage/taktician/cmd/internal/play"
)

func main() {
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	// subcommands.Register(subcommands.CommandsCommand(), "")

	subcommands.Register(&play.Command{}, "")

	subcommands.Register(&analyze.Command{}, "analysis")
	subcommands.Register(&canonicalize.Command{}, "analysis")

	flag.Parse()
	ctx := context.Background()
	os.Exit(int(subcommands.Execute(ctx)))
}
