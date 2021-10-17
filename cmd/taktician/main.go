package main

import (
	"context"
	"flag"
	"os"

	"github.com/google/subcommands"
	"github.com/nelhage/taktician/cmd/internal/analyze"
	"github.com/nelhage/taktician/cmd/internal/canonicalize"
	"github.com/nelhage/taktician/cmd/internal/gencorpus"
	"github.com/nelhage/taktician/cmd/internal/genopenings"
	"github.com/nelhage/taktician/cmd/internal/importptn"
	"github.com/nelhage/taktician/cmd/internal/openings"
	"github.com/nelhage/taktician/cmd/internal/play"
	"github.com/nelhage/taktician/cmd/internal/playtak"
	"github.com/nelhage/taktician/cmd/internal/selfplay"
	"github.com/nelhage/taktician/cmd/internal/serve"
	"github.com/nelhage/taktician/cmd/internal/tei"
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
	subcommands.Register(&tei.Command{}, "")

	subcommands.Register(&genopenings.Command{}, "")
	subcommands.Register(&gencorpus.Command{}, "")
	subcommands.Register(&openings.Command{}, "")
	subcommands.Register(&canonicalize.Command{}, "")

	subcommands.Register(&importptn.Command{}, "")

	flag.Parse()
	ctx := context.Background()
	os.Exit(int(subcommands.Execute(ctx)))
}
