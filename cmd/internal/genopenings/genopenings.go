package genopenings

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"

	"github.com/google/subcommands"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/symmetry"
	"github.com/nelhage/taktician/tak"
)

type Command struct {
	seed  int64
	size  int
	depth int
	n     int

	rand *rand.Rand
}

func (*Command) Name() string     { return "genopenings" }
func (*Command) Synopsis() string { return "Generate a set of opening positions" }
func (*Command) Usage() string {
	return `genopenings [flags]
`
}

func (c *Command) SetFlags(flags *flag.FlagSet) {
	flags.IntVar(&c.size, "size", 5, "what size to analyze")
	flags.IntVar(&c.depth, "depth", 2, "generate openings to what depth")
	flags.IntVar(&c.n, "n", 100, "generate how many openings")
	flags.Int64Var(&c.seed, "seed", 0, "Random seed")
}

func (c *Command) Execute(ctx context.Context, flag *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	c.rand = rand.New(rand.NewSource(c.seed))
	init := tak.New(tak.Config{Size: c.size})
	var positions []*tak.Position
	seen := make(map[uint64]*tak.Position)

generate:
	for len(positions) < c.n {
		pos := c.generate(init, c.depth)
		syms, _ := symmetry.Symmetries(pos)
		for _, sym := range syms {
			if got, ok := seen[sym.P.Hash()]; ok {
				if !got.Equal(sym.P) {
					log.Fatalf("hash collision seen=%q new=%q", ptn.FormatTPS(got), ptn.FormatTPS(sym.P))
				}
				continue generate
			}
		}
		seen[pos.Hash()] = pos
		positions = append(positions, pos)
	}

	for _, pos := range positions {
		fmt.Println(ptn.FormatTPS(pos))
	}
	return subcommands.ExitSuccess
}

func (c *Command) generate(pos *tak.Position, depth int) *tak.Position {
	var buf [100]tak.Move
	for d := 0; d < depth; d++ {
		moves := pos.AllMoves(buf[:0])
		for {
			m := moves[c.rand.Intn(len(moves))]
			n, e := pos.Move(m)
			if e != nil {
				continue
			}
			pos = n
			break
		}
	}
	return pos
}
