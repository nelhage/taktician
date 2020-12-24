package tei

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

type Engine struct {
	ConfigFactory func(size int) ai.MinimaxConfig

	in  *bufio.Reader
	out io.Writer

	mm   *ai.MinimaxAI
	pos  *tak.Position
	size int
}

func NewEngine(in io.Reader, out io.Writer) *Engine {
	return &Engine{
		in:  bufio.NewReader(in),
		out: out,
	}
}

func (e *Engine) Run(ctx context.Context) error {
	for {
		line, err := e.in.ReadString('\n')
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		words := strings.Fields(line)
		switch words[0] {
		case "tei":
			fmt.Fprintln(e.out, "id name Taktician")
			fmt.Fprintln(e.out, "id author Nelson Elhage")
			fmt.Fprintln(e.out, "teiok")
		case "quit":
			break
		case "teinewgame":
			e.mm = nil
			e.pos = nil
			if len(words) > 1 {
				e.size, err = strconv.Atoi(words[1])
				if err != nil || e.size < 3 || e.size > 8 {
					return fmt.Errorf("Bad size: %s", words[1])
				}
			} else {
				e.size = 5
			}
			break
		case "position":
			e.pos, err = parsePosition(e.size, words)
			if err != nil {
				return fmt.Errorf("error parsing position: %w\n", err)
			}
			break
		case "go":
			if err := e.analyze(ctx, words); err != nil {
				log.Printf("error in go: %v\n", err)
				break
			}
			break
		case "stop":
			break
		case "isready":
			fmt.Fprintln(e.out, "readyok")
		default:
			return fmt.Errorf("Unknown command: %q", line)
		}
	}
}

func parsePosition(size int, words []string) (*tak.Position, error) {
	var pos *tak.Position
	words = words[1:]
	if len(words) == 0 {
		return nil, errors.New("not enough arguments")
	}
	switch words[0] {
	case "startpos":
		words = words[1:]
		pos = tak.New(tak.Config{Size: size})
	case "tps":
		// tps A B C
		if len(words) < 4 {
			return nil, errors.New("position tps: not enough arguments")
		}
		var err error
		pos, err = ptn.ParseTPS(strings.Join(words[1:4], " "))
		if err != nil {
			return nil, fmt.Errorf("Parse TPS: %w", err)
		}
		words = words[4:]
		if pos.Size() != size {
			return nil, fmt.Errorf("tps has wrong size: got %d, configured for %d", pos.Size(), size)
		}
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

func calcBudget(movetime time.Duration, gametime time.Duration, inc time.Duration) time.Duration {
	var budget time.Duration
	if gametime != 0 {
		budget = (gametime + inc) / 10
	}
	if movetime > 0 && (budget == 0 || movetime < budget) {
		budget = movetime
	}
	return budget
}

func (e *Engine) analyze(ctx context.Context, words []string) error {
	if e.pos == nil {
		return errors.New("No position provided")
	}
	if e.mm == nil {
		var cfg ai.MinimaxConfig
		if e.ConfigFactory != nil {
			cfg = e.ConfigFactory(e.size)
		} else {
			cfg = ai.MinimaxConfig{
				Size: e.size,
			}
		}
		e.mm = ai.NewMinimax(cfg)
	}
	words = words[1:]
	var movetime time.Duration
	var tc TimeControl
	timeArgs := map[string]*time.Duration{
		"movetime": &movetime,
		"wtime":    &tc.White,
		"btime":    &tc.Black,
		"winc":     &tc.WInc,
		"binc":     &tc.BInc,
	}
	for len(words) > 0 {
		opt := words[0]
		if len(words) == 1 {
			return fmt.Errorf("%s: expected arg", opt)
		}
		arg := words[1]
		words = words[2:]
		if dur, ok := timeArgs[opt]; ok {
			if ms, err := strconv.ParseUint(arg, 10, 64); err == nil {
				*dur = time.Millisecond * time.Duration(ms)
			} else {
				return fmt.Errorf("%s: cannot parse value: %q", opt, arg)
			}
		} else {
			return fmt.Errorf("Unknown option: %s", opt)
		}
	}
	var tm, inc time.Duration
	if e.pos.ToMove() == tak.White {
		tm, inc = tc.White, tc.WInc
	} else {
		tm, inc = tc.Black, tc.BInc
	}

	if movetime > 0 || tm > 0 {
		budget := calcBudget(movetime, tm, inc)
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, budget)
		defer cancel()
	}

	pv, val, stats := e.mm.Analyze(ctx, e.pos)
	var pvs strings.Builder
	for _, m := range pv {
		pvs.WriteString(" ")
		pvs.WriteString(ptn.FormatMove(m))
	}
	fmt.Fprintf(e.out, "info depth %d time %d nodes %d score cp %d pv%s\n",
		stats.Depth,
		stats.Elapsed/time.Millisecond,
		stats.Visited,
		val,
		pvs.String(),
	)
	fmt.Fprintf(e.out, "bestmove %s\n", ptn.FormatMove(pv[0]))
	return nil
}
