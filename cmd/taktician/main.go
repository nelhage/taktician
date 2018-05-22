package main

import (
	"context"
	"flag"
	"os"

	"github.com/google/subcommands"
	"github.com/nelhage/taktician/cmd/internal/analyze"
	"github.com/nelhage/taktician/cmd/internal/canonicalize"
	"github.com/nelhage/taktician/cmd/internal/importptn"
	"github.com/nelhage/taktician/cmd/internal/logger"
	"github.com/nelhage/taktician/cmd/internal/openings"
	"github.com/nelhage/taktician/cmd/internal/play"
	"github.com/nelhage/taktician/cmd/internal/playtak"
	"github.com/nelhage/taktician/cmd/internal/selfplay"
	"github.com/nelhage/taktician/cmd/internal/serve"
)

func main() {
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	// subcommands.Register(subcommands.CommandsCommand(), "")

	subcommands.Register(&analyze.Command{}, "")
	subcommands.Register(&selfplay.Command{}, "")
	subcommands.Register(&playtak.Command{}, "")
	subcommands.Register(&serve.Command{}, "")
	subcommands.Register(&play.Command{}, "")

	subcommands.Register(&logger.Command{}, "")
	subcommands.Register(&openings.Command{}, "")
	subcommands.Register(&canonicalize.Command{}, "")

	subcommands.Register(&importptn.Command{}, "")

	flag.Parse()
	ctx := context.Background()
	os.Exit(int(subcommands.Execute(ctx)))
}
