package badmoves

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"os"

	"log"

	"github.com/google/subcommands"
	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/cmd/internal/opt"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

type Command struct {
	mm opt.Minimax

	white, black bool
	delta        int64

	csv string
}

func (*Command) Name() string     { return "badmoves" }
func (*Command) Synopsis() string { return "Find bad moves in a PTN file" }
func (*Command) Usage() string {
	return `analyze [options] FILE.ptn
`
}

func (c *Command) SetFlags(flags *flag.FlagSet) {
	c.mm.AddFlags(flags)

	flags.BoolVar(&c.white, "white", false, "Only evaluate moves by white")
	flags.BoolVar(&c.black, "black", false, "Only evaluate moves by black")
	flags.Int64Var(&c.delta, "delta", 1000, "Lost points to count as a bad move")

	flags.StringVar(&c.csv, "csv", "", "Write out CSV file")

}

func (c *Command) Execute(ctx context.Context, flag *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.mm.Depth == 0 {
		log.Printf("-depth is required for %s", c.Name())
		return subcommands.ExitUsageError
	}

	parsed, e := ptn.ParseFile(flag.Arg(0))
	if e != nil {
		log.Fatal("parse:", e)
	}

	p, e := parsed.InitialPosition()
	if e != nil {
		log.Fatal("bad ptn:", e)
	}

	var csvw *csv.Writer
	if c.csv != "" {
		fh, err := os.Create(c.csv)
		if err != nil {
			log.Fatalf("open(%q): %v", c.csv, err)
		}
		defer fh.Close()
		csvw = csv.NewWriter(fh)
		defer csvw.Flush()

		csvw.Write([]string{
			"path",
			"ply",
			"tps",
			"move",
			"value",
			"pv",
			"pv_value",
		})
	}

	if c.white && c.black {
		c.white = false
		c.black = false
	}

	cfg := c.mm.BuildConfig(p.Size())
	mainEngine := ai.NewMinimax(cfg)
	cfg.Depth -= 1
	secondaryEngine := ai.NewMinimax(cfg)

	for _, path := range flag.Args() {
		parsed, e := ptn.ParseFile(path)
		if e != nil {
			log.Fatalf("parse %q: %v", path, e)
		}

		it := parsed.Iterator()
		for it.Next() {
			p := it.Position()
			m := it.PeekMove()
			if m.Type == tak.Pass {
				break
			}
			if over, _ := p.GameOver(); over {
				break
			}

			if (c.white && p.ToMove() == tak.Black) || (c.black && p.ToMove() == tak.White) {
				continue
			}

			pNext, e := p.Move(m)
			if e != nil {
				log.Fatalf("move %d: bad move: %s", it.PTNMove(), ptn.FormatMove(m))
			}

			if over, _ := pNext.GameOver(); over {
				break
			}

			rootPV, rootVal, _ := mainEngine.Analyze(ctx, p)
			if rootPV[0] == m {
				continue
			}
			_, moveVal, _ := secondaryEngine.Analyze(ctx, pNext)
			moveVal = -moveVal
			if moveVal < (rootVal - c.delta) {
				log.Printf("bad move path=%q ply=%d tps=%q move=%s score=%d pv=%s pvscore=%d",
					path,
					p.MoveNumber(),
					ptn.FormatTPS(p),
					ptn.FormatMove(m),
					moveVal,
					ptn.FormatMove(rootPV[0]),
					rootVal,
				)
				if csvw != nil {
					csvw.Write([]string{
						path,
						fmt.Sprintf("%d", p.MoveNumber()),
						ptn.FormatTPS(p),
						ptn.FormatMove(m),
						fmt.Sprintf("%d", moveVal),
						ptn.FormatMove(rootPV[0]),
						fmt.Sprintf("%d", rootVal),
					})
				}
			}
		}
	}

	return subcommands.ExitSuccess
}
