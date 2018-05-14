package main

import (
	"context"
	"flag"
	"os"

	"github.com/google/subcommands"
	"github.com/nelhage/taktician/cmd/internal/analyze"
	"github.com/nelhage/taktician/cmd/internal/canonicalize"
	"github.com/nelhage/taktician/cmd/internal/logger"
	"github.com/nelhage/taktician/cmd/internal/openings"
	"github.com/nelhage/taktician/cmd/internal/play"
	"github.com/nelhage/taktician/cmd/internal/selfplay"
)

func main() {
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	// subcommands.Register(subcommands.CommandsCommand(), "")

	subcommands.Register(&logger.Command{}, "")
	subcommands.Register(&openings.Command{}, "")

	subcommands.Register(&play.Command{}, "")

	subcommands.Register(&analyze.Command{}, "")
	subcommands.Register(&selfplay.Command{}, "")

	subcommands.Register(&canonicalize.Command{}, "")

	flag.Parse()
	ctx := context.Background()
	os.Exit(int(subcommands.Execute(ctx)))
}
