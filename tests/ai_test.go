package tests

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/cli"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

var debug = flag.Int("debug", 0, "debug level")
var overrideConfig = flag.String("config", "", "override config")
var dumpPerf = flag.Bool("debug-perf", false, "debug perf")
var zooPath = flag.String("zoo", "../testdata/ai", "path to test zoo")

type moveSpec struct {
	number    int
	color     tak.Color
	maxEval   uint64
	badMoves  []tak.Move
	goodMoves []tak.Move
}

type TestCase struct {
	p    *ptn.PTN
	id   string
	name string

	cfg ai.MinimaxConfig

	moves []moveSpec

	speed string

	limit time.Duration
}

func TestAIRegression(t *testing.T) {
	ptns, e := readPTNs(*zooPath)
	if e != nil {
		panic(e)
	}
	cases := []*TestCase{}
	for path, p := range ptns {
		tc, e := preparePTN(path, p)
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

func preparePTN(path string, p *ptn.PTN) (*TestCase, error) {
	tc := TestCase{
		p:     p,
		cfg:   ai.MinimaxConfig{Depth: 5, Seed: 1},
		limit: time.Minute,
		name:  strings.TrimSuffix(path, ".ptn"),
	}
	if *overrideConfig != "" {
		e := json.Unmarshal([]byte(*overrideConfig), &tc.cfg)
		if e != nil {
			return nil, fmt.Errorf("bad -config: %s", e)
		}
	}
	var e error
	var spec *moveSpec
	for _, t := range p.Tags {
		if t.Value == "" {
			continue
		}
		switch t.Name {
		case "Move":
			bits := strings.Split(t.Value, " ")
			tc.moves = append(tc.moves, moveSpec{})
			spec = &tc.moves[len(tc.moves)-1]
			spec.number, e = strconv.Atoi(bits[0])
			if e != nil {
				return nil, fmt.Errorf("%s: bad move: `%s`", path, t.Value)
			}
			if len(bits) > 1 {
				switch bits[1] {
				case "white":
					spec.color = tak.White
				case "black":
					spec.color = tak.Black
				default:
					return nil, fmt.Errorf("%s: bad color: `%s`", path, t.Value)
				}
			}
		case "MaxEval":
			if spec == nil {
				return nil, fmt.Errorf("%s: MaxEval before Move", path)
			}
			spec.maxEval, e = strconv.ParseUint(t.Value, 10, 64)
			if e != nil {
				return nil, fmt.Errorf("%s: bad MaxEval: %s", path, t.Value)
			}
		case "Depth":
			tc.cfg.Depth, e = strconv.Atoi(t.Value)
			if e != nil {
				return nil, fmt.Errorf("%s: bad depth: %s", path, t.Value)
			}
		case "BadMove":
			if spec == nil {
				return nil, fmt.Errorf("%s: BadMove before Move", path)
			}
			move, e := ptn.ParseMove(t.Value)
			if e != nil {
				return nil, fmt.Errorf("%s: bad move: `%s': %v", path, t.Value, e)
			}
			spec.badMoves = append(spec.badMoves, move)
		case "GoodMove":
			if spec == nil {
				return nil, fmt.Errorf("%s: GoodMove before Move", path)
			}
			move, e := ptn.ParseMove(t.Value)
			if e != nil {
				return nil, fmt.Errorf("%s: bad move: `%s': %v", path, t.Value, e)
			}
			spec.goodMoves = append(spec.goodMoves, move)
		case "Limit":
			tc.limit, e = time.ParseDuration(t.Value)
			if e != nil {
				return nil, fmt.Errorf("%s: bad limit: `%s`: %v", path, t.Value, e)
			}
		case "Seed":
			tc.cfg.Seed, e = strconv.ParseInt(t.Value, 10, 64)
			if e != nil {
				return nil, fmt.Errorf("%s: bad MaxEval: %s", path, t.Value)
			}
		case "Speed":
			tc.speed = t.Value
		case "Id":
			tc.id = t.Value
		case "Name":
			tc.name = t.Value
		case "Size":
			sz, e := strconv.ParseInt(t.Value, 10, 64)
			if e != nil {
				return nil, fmt.Errorf("%s: bad Size: %v", path, e)
			}
			tc.cfg.Size = int(sz)
		}
	}
	return &tc, nil
}

func runTest(t *testing.T, tc *TestCase) {
	name := ""
	if tc.id != "" {
		name = fmt.Sprintf("[%s]", tc.id)
	}
	name = fmt.Sprintf("%s%s", name, tc.name)
	t.Logf("considering %s...", name)
	cfg := tc.cfg
	cfg.Debug = *debug
	ai := ai.NewMinimax(cfg)
	for _, spec := range tc.moves {
		t.Logf("evaluating %d. %s", spec.number, spec.color)
		p, e := tc.p.PositionAtMove(spec.number, spec.color)
		if e != nil {
			t.Errorf("!! %s: find move: %v", name, e)
			return
		}
		var buf bytes.Buffer
		cli.RenderBoard(nil, &buf, p)
		t.Log(buf.String())
		start := time.Now()
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(tc.limit))
		pvs, v, st := ai.AnalyzeAll(ctx, p)
		cancel()
		elapsed := time.Now().Sub(start)
		if *dumpPerf {
			log.Printf("%s move=%d color=%s depth=%d evaluated=%d time=%s",
				tc.id, spec.number, spec.color, st.Depth, st.Evaluated, elapsed,
			)
		}
		if len(pvs) == 0 {
			t.Errorf("!! %s: did not return a move!", name)
			return
		}
		for _, pv := range pvs {
			var ms []string
			for _, m := range pv {
				ms = append(ms, ptn.FormatMove(&m))
			}
			t.Logf("ai: pv=[%s] value=%v evaluated=%d", strings.Join(ms, " "), v, st.Evaluated)
			_, e = p.Move(&pv[0])
			if e != nil {
				t.Errorf("!! %s: illegal move: `%s'", name, ptn.FormatMove(&pv[0]))
			}
			for _, m := range spec.badMoves {
				if pv[0].Equal(&m) {
					t.Errorf("!! %s: bad move: `%s'", name, ptn.FormatMove(&pv[0]))
				}
			}
			found := false
			for _, m := range spec.goodMoves {
				if pv[0].Equal(&m) {
					found = true
					break
				}
			}
			if len(spec.goodMoves) != 0 && !found {
				t.Errorf("!! %s is not an allowed good move", ptn.FormatMove(&pv[0]))
			}
		}
		if spec.maxEval != 0 && st.Evaluated > spec.maxEval {
			t.Errorf("!! %s: evaluated %d > %d positions",
				name, st.Evaluated, spec.maxEval)
		}
	}
}
