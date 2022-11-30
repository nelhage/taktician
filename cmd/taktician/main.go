package main

import (
	"context"
	"flag"
	"log"
	"os"
	"runtime/pprof"

	"github.com/google/subcommands"
	"github.com/nelhage/taktician/cmd/internal/analyze"
	"github.com/nelhage/taktician/cmd/internal/badmoves"
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

func innerMain() int {
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	// subcommands.Register(subcommands.CommandsCommand(), "")

	subcommands.Register(&analyze.Command{}, "")
	subcommands.Register(&selfplay.Command{}, "")
	subcommands.Register(&playtak.Command{}, "")
	subcommands.Register(&serve.Command{}, "")
	subcommands.Register(&play.Command{}, "")
	subcommands.Register(&tei.Command{}, "")
	subcommands.Register(&badmoves.Command{}, "")

	subcommands.Register(&genopenings.Command{}, "")
	subcommands.Register(&openings.Command{}, "")
	subcommands.Register(&canonicalize.Command{}, "")
	subcommands.Register(&gencorpus.Command{}, "")

	subcommands.Register(&importptn.Command{}, "")

	var cpuProfile, memProfile string

	flag.StringVar(&cpuProfile, "cpuprofile", "", "write CPU profile")
	flag.StringVar(&memProfile, "memprofile", "", "write memory profile")

	flag.Parse()

	if cpuProfile != "" {
		f, e := os.OpenFile(cpuProfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if e != nil {
			log.Fatalf("open cpu-profile: %s: %v", cpuProfile, e)
		}
		pprof.StartCPUProfile(f)
		defer f.Close()
		defer pprof.StopCPUProfile()
	}
	if memProfile != "" {
		f, e := os.OpenFile(memProfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if e != nil {
			log.Fatalf("open memory profile: %s: %v", cpuProfile, e)
		}
		defer func() {
			pprof.Lookup("allocs").WriteTo(f, 0)
			f.Close()
		}()
	}

	ctx := context.Background()
	return int(subcommands.Execute(ctx))
}

func main() {
	os.Exit(innerMain())
}
