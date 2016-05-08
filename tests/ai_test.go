package tests

import (
	"bytes"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/cli"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

var debug = flag.Int("debug", 0, "debug level")

type TestCase struct {
	p          *ptn.PTN
	id         string
	moveNumber int
	color      tak.Color

	cfg ai.MinimaxConfig

	maxEval  uint64
	badMoves []tak.Move

	speed string

	limit time.Duration
}

func TestAIRegression(t *testing.T) {
	ptns, e := readPTNs("data/ai")
	if e != nil {
		panic(e)
	}
	cases := []*TestCase{}
	for _, p := range ptns {
		tc, e := preparePTN(p)
		if e != nil {
			t.Errorf("prepare ptn: %v", e)
			continue
		}
		cases = append(cases, tc)
	}

	for _, tc := range cases {
		runTest(t, tc)
	}
}

func preparePTN(p *ptn.PTN) (*TestCase, error) {
	tc := TestCase{
		p:     p,
		cfg:   ai.MinimaxConfig{Depth: 5},
		limit: time.Minute,
	}
	var e error
	for _, t := range p.Tags {
		if t.Value == "" {
			continue
		}
		switch t.Name {
		case "Move":
			bits := strings.Split(t.Value, " ")
			tc.moveNumber, e = strconv.Atoi(bits[0])
			if e != nil {
				return nil, fmt.Errorf("bad move: `%s`", t.Value)
			}
			if len(bits) > 1 {
				switch bits[1] {
				case "white":
					tc.color = tak.White
				case "black":
					tc.color = tak.Black
				default:
					return nil, fmt.Errorf("bad color: `%s`", t.Value)
				}
			}
		case "MaxEval":
			tc.maxEval, e = strconv.ParseUint(t.Value, 10, 64)
			if e != nil {
				return nil, fmt.Errorf("bad MaxEval: %s", t.Value)
			}
		case "Depth":
			tc.cfg.Depth, e = strconv.Atoi(t.Value)
			if e != nil {
				return nil, fmt.Errorf("bad depth: %s", t.Value)
			}
		case "BadMove":
			move, e := ptn.ParseMove(t.Value)
			if e != nil {
				return nil, fmt.Errorf("bad move: `%s': %v", t.Value, e)
			}
			tc.badMoves = append(tc.badMoves, move)
		case "Limit":
			tc.limit, e = time.ParseDuration(t.Value)
			if e != nil {
				return nil, fmt.Errorf("bad limit: `%s`: %v", t.Value, e)
			}
		case "Seed":
			tc.cfg.Seed, e = strconv.ParseInt(t.Value, 10, 64)
			if e != nil {
				return nil, fmt.Errorf("bad MaxEval: %s", t.Value)
			}
		case "Speed":
			tc.speed = t.Value
		case "Id":
			tc.id = t.Value
		}
	}
	return &tc, nil
}

func runTest(t *testing.T, tc *TestCase) {
	t.Logf("considering `%s'...", tc.id)
	p, e := tc.p.PositionAtMove(tc.moveNumber, tc.color)
	if e != nil {
		t.Errorf("%s: find move: %v", tc.id, e)
		return
	}
	var buf bytes.Buffer
	cli.RenderBoard(&buf, p)
	t.Log(buf.String())
	cfg := tc.cfg
	cfg.Size = p.Size()
	cfg.Debug = *debug
	ai := ai.NewMinimax(cfg)
	pv, v, st := ai.Analyze(p, tc.limit)
	if len(pv) == 0 {
		t.Errorf("%s: did not return a move!", tc.id)
		return
	}
	var ms []string
	for _, m := range pv {
		ms = append(ms, ptn.FormatMove(&m))
	}
	t.Logf("ai: pv=[%s] value=%v", strings.Join(ms, " "), v)
	_, e = p.Move(&pv[0])
	if e != nil {
		t.Errorf("%s: illegal move: `%s'", tc.id, ptn.FormatMove(&pv[0]))
	}
	for _, m := range tc.badMoves {
		if pv[0].Equal(&m) {
			t.Errorf("%s: bad move: `%s'", tc.id, ptn.FormatMove(&pv[0]))
		}
	}
	if tc.maxEval != 0 && st.Evaluated > tc.maxEval {
		t.Errorf("%s: evaluated %d > %d positions",
			tc.id, st.Evaluated, tc.maxEval)
	}
}
