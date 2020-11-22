package engine

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/subcommands"
	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

type Command struct {
	mm   *ai.MinimaxAI
	pos  *tak.Position
	size int
}

func (*Command) Name() string     { return "engine" }
func (*Command) Synopsis() string { return "Launch Taktician in UCI-like engine mode" }
func (*Command) Usage() string {
	return `engine
Launch the engine in a UCI-like mode, suitable for being
driven by an external GUI or controller.`
}

func (c *Command) SetFlags(fs *flag.FlagSet) {
	fs.IntVar(&c.size, "size", 5, "Board size for engine")
}

func (c *Command) Execute(ctx context.Context, flag *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	rdr := bufio.NewReader(os.Stdin)
	for {
		line, err := rdr.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("IO error: %v", err)
			return subcommands.ExitFailure
		}
		line = line[:len(line)-1]
		if line == "" {
			continue
		}
		words := strings.Split(line, " ")
		switch words[0] {
		case "uti":
			fmt.Println("id name Taktician")
			fmt.Println("id author Nelson Elhage")
			fmt.Println("utiok")
		case "quit":
			break
		case "utinewgame":
			c.mm = nil
			c.pos = nil
			break
		case "position":
			var err error
			c.pos, err = parsePosition(c.size, words)
			if err != nil {
				log.Printf("error parsing position: %v\n", err)
				break
			}
			break
		case "go":
			if err := c.analyze(ctx, words); err != nil {
				log.Printf("error in go: %v\n", err)
				break
			}
			break
		case "stop":
			break
		case "isready":
			log.Println("readyok")
		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s", line)
			break
		}
	}

	return subcommands.ExitSuccess
}

func parsePosition(size int, words []string) (*tak.Position, error) {
	var pos *tak.Position
	words = words[1:]
	if len(words) == 0 {
		return nil, errors.New("not enoug arguments")
	}
	switch words[0] {
	case "startpos":
		words = words[1:]
		pos = tak.New(tak.Config{Size: size})
	case "tps":
		if len(words) < 2 {
			return nil, errors.New("position ptn: not enough arguments")
		}
		var err error
		pos, err = ptn.ParseTPS(words[1])
		if err != nil {
			return nil, fmt.Errorf("Parse TPS: %w", err)
		}
		words = words[2:]
	default:
		return nil, fmt.Errorf("Unknown initial position: %q", words[0])
	}
	if len(words) == 0 {
		return pos, nil
	}
	if words[0] != "moves" {
		return nil, errors.New("position: expected `moves'")
	}
	words = words[1:]
	for _, w := range words {
		move, err := ptn.ParseMove(w)
		if err != nil {
			return nil, fmt.Errorf("Parse move %q: %w", w, err)
		}
		pos, err = pos.Move(move)
		if err != nil {
			return nil, fmt.Errorf("Move %q: %w", w, err)
		}
	}
	return pos, nil
}

func (c *Command) analyze(ctx context.Context, words []string) error {
	if c.pos == nil {
		return errors.New("No position provided")
	}
	if c.mm == nil {
		c.mm = ai.NewMinimax(ai.MinimaxConfig{
			Size: c.size,
		})
	}
	words = words[1:]
	if len(words) != 2 || words[0] != "movetime" {
		return errors.New("expected <movetime> N")
	}
	ms, err := strconv.ParseUint(words[1], 10, 64)
	if err != nil {
		return fmt.Errorf("bad ms: %v", words[1])
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(ms)*time.Millisecond)
	defer cancel()

	pv, val, stats := c.mm.Analyze(ctx, c.pos)
	var pvs strings.Builder
	for _, m := range pv {
		pvs.WriteString(" ")
		pvs.WriteString(ptn.FormatMove(m))
	}
	fmt.Printf("info depth %d time %d nodes %d score cp %d pv%s",
		stats.Depth,
		stats.Elapsed/time.Millisecond,
		stats.Visited,
		val,
		pvs.String(),
	)
	fmt.Printf("bestmove %s", ptn.FormatMove(pv[0]))
	return nil
}
