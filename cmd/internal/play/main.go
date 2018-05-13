package play

import (
	"bufio"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"context"

	"github.com/google/subcommands"
	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/ai/mcts"
	"github.com/nelhage/taktician/cli"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

type Command struct {
	white string
	black string
	size  int
	debug int
	limit time.Duration
	out   string

	unicode bool
}

func (*Command) Name() string     { return "play" }
func (*Command) Synopsis() string { return "Play Tak from the command line" }
func (*Command) Usage() string {
	return `play

Play Tak on the command-line, against a human or AI.
`
}

func (c *Command) SetFlags(flags *flag.FlagSet) {
	flags.StringVar(&c.white, "white", "human", "white player")
	flags.StringVar(&c.black, "black", "human", "white player")
	flags.IntVar(&c.size, "size", 5, "game size")
	flags.IntVar(&c.debug, "debug", 0, "debug level")
	flags.DurationVar(&c.limit, "limit", time.Minute, "ai time limit")
	flags.StringVar(&c.out, "out", "", "write ptn to file")

	flags.BoolVar(&c.unicode, "unicode", false, "render board with utf8 glyphs")
}

func (c *Command) Execute(ctx context.Context, flag *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	in := bufio.NewReader(os.Stdin)
	st := &cli.CLI{
		Config: tak.Config{Size: c.size},
		Out:    os.Stdout,
		White:  c.parsePlayer(in, c.white),
		Black:  c.parsePlayer(in, c.black),
		Glyphs: glyphs(c.unicode),
	}
	st.Play()
	if c.out != "" {
		p := &ptn.PTN{}
		p.Tags = []ptn.Tag{
			{Name: "Size", Value: strconv.Itoa(c.size)},
			{Name: "Player1", Value: c.white},
			{Name: "Player2", Value: c.black},
		}
		p.AddMoves(st.Moves())
		ioutil.WriteFile(c.out, []byte(p.Render()), 0644)
	}

	return subcommands.ExitSuccess
}

func glyphs(unicode bool) *cli.Glyphs {
	if unicode {
		return &cli.UnicodeGlyphs
	}
	return &cli.DefaultGlyphs
}

type aiWrapper struct {
	limit time.Duration
	p     ai.TakPlayer
}

func (a *aiWrapper) GetMove(p *tak.Position) tak.Move {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(a.limit))
	defer cancel()
	return a.p.GetMove(ctx, p)
}

func (c *Command) parsePlayer(in *bufio.Reader, s string) cli.Player {
	if s == "human" {
		return cli.NewCLIPlayer(os.Stdout, in)
	}
	if s == "rand" {
		return &aiWrapper{c.limit, ai.NewRandom(0)}
	}
	if strings.HasPrefix(s, "rand") {
		var seed int64
		if len(s) > len("rand") {
			i, err := strconv.Atoi(s[len("rand:"):])
			if err != nil {
				log.Fatal(err)
			}
			seed = int64(i)
		}
		return &aiWrapper{c.limit, ai.NewRandom(seed)}
	}
	if strings.HasPrefix(s, "minimax") {
		var depth = 3
		if len(s) > len("minimax") {
			i, err := strconv.Atoi(s[len("minimax:"):])
			if err != nil {
				log.Fatal(err)
			}
			depth = i
		}
		p := ai.NewMinimax(ai.MinimaxConfig{
			Size:  c.size,
			Depth: depth,
			Debug: c.debug,
		})
		return &aiWrapper{c.limit, p}
	}
	if strings.HasPrefix(s, "mcts") {
		var limit = 30 * time.Second
		if len(s) > len("mcts") {
			var err error
			limit, err = time.ParseDuration(s[len("mcts:"):])
			if err != nil {
				log.Fatal(err)
			}
		}
		p := mcts.NewMonteCarlo(mcts.MCTSConfig{
			Limit: c.limit,
			Debug: c.debug,
			Size:  c.size,
		})
		return &aiWrapper{limit, p}
	}
	log.Fatalf("unparseable player: %s", s)
	return nil
}
