package ai

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/nelhage/taktician/bitboard"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

func TestEvaluateWinner(t *testing.T) {
	cases := []struct {
		tps      string
		min, max int64
	}{
		{
			`x3,2,x/2S,1,2121,2221C,1/1,1,x,111112C,x/1,x,2,2,x/1,1,22111112S,1,2221 2 35`,
			MinEval, -WinThreshold,
		},
		{
			`x4,1/x4,1/x3,2,1/x3,2,1/2,x3,1 1 6`,
			WinThreshold, MaxEval,
		},
		{
			`x4,1/x4,1/x3,2,1/x3,2,x/2,x4 1 4`,
			0, 0,
		},
	}
	c := bitboard.Precompute(5)
	for i, tc := range cases {
		p, e := ptn.ParseTPS(tc.tps)
		if e != nil {
			t.Errorf("%d: tps: %v", i, e)
			continue
		}
		eval := EvaluateWinner(&c, p)
		if eval < tc.min || eval > tc.max {
			t.Errorf("%d: eval=%d (not in [%d,%d])", i, eval, tc.min, tc.max)
		}
		termEval := evaluateTerminal(p, &defaultTerminal, p.WinDetails().Winner)
		if termEval < tc.min || termEval > tc.max {
			t.Errorf("%d: evalTerminal=%d (not in [%d,%d])", i, termEval, tc.min, tc.max)
		}
	}
}

func formatpvs(pvs [][]tak.Move) string {
	var ptns []string
	for _, pv := range pvs {
		ptns = append(ptns, formatpv(pv))
	}
	return fmt.Sprintf("%v", ptns)
}

func TestEvaluateTerminal(t *testing.T) {
	p, e := ptn.ParseTPS(`x,2,2,11121S,1,12S/2S,1,2S,1,2S,2/2,1,2,1212C,1,1/1,21,1,21C,2S,21/2,1,2,2,2,1/2,2,1,1,2,1 1 31`)
	if e != nil {
		panic(e)
	}

	ai := NewMinimax(MinimaxConfig{Size: p.Size(), Depth: 5})
	pvs, _, _ := ai.AnalyzeAll(context.Background(), p)
	if len(pvs) != 1 {
		t.Fatalf("returned too many moves: %q", formatpvs(pvs))
	}
	f := ptn.FormatMove(&pvs[0][0])
	if f != "4d6<112" {
		t.Errorf("returned wrong move: %s", f)
	}

	ai = NewMinimax(MinimaxConfig{
		Size:  p.Size(),
		Depth: 5,
		Evaluate: MakeEvaluator(p.Size(), &Weights{
			Terminal: TerminalWeights{
				Flats:    0,
				Reserves: 1,
			},
		}),
	})
	pvs, _, _ = ai.AnalyzeAll(context.Background(), p)
	if len(pvs) != 4 {
		t.Fatal("did not find all slides")
	}
	for _, pv := range pvs {
		if !pv[0].IsSlide() {
			t.Fatal("placed a reserve!")
		}
	}
}

func benchmarkEval(b *testing.B, tps string) {
	p, e := ptn.ParseTPS(tps)
	if e != nil {
		b.Fatal("tps:", e)
	}
	c := bitboard.Precompute(uint(p.Size()))
	w := DefaultWeights[p.Size()]
	w.Potential = 100
	w.Threat = 300
	eval := MakeEvaluator(p.Size(), &w)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eval(&c, p)
	}
}

func BenchmarkEvalEarlyGame(b *testing.B) {
	benchmarkEval(b, `x5/x3,2,x/x2,1C,1,2/x2,2,1,1/2,x2,2C,1 1 6`)
}

func BenchmarkEvalMidGame(b *testing.B) {
	benchmarkEval(b, `x3,2,x/x4,12/1,1,x,1,21C/x,1,x,12111112C,2/2,x,22121,x,2 2 20`)
}

func board(tpl string, who tak.Color) (*tak.Position, error) {
	lines := strings.Split(strings.Trim(tpl, " \n"), "\n")
	var pieces [][]tak.Square
	for _, l := range lines {
		bits := strings.Split(l, " ")
		var row []tak.Square
		for _, p := range bits {
			switch p {
			case "W":
				row = append(row, tak.Square{tak.MakePiece(tak.White, tak.Flat)})
			case "B":
				row = append(row, tak.Square{tak.MakePiece(tak.Black, tak.Flat)})
			case "WC":
				row = append(row, tak.Square{tak.MakePiece(tak.White, tak.Capstone)})
			case "BC":
				row = append(row, tak.Square{tak.MakePiece(tak.Black, tak.Capstone)})
			case "WS":
				row = append(row, tak.Square{tak.MakePiece(tak.White, tak.Standing)})
			case "BS":
				row = append(row, tak.Square{tak.MakePiece(tak.Black, tak.Standing)})
			case ".":
				row = append(row, tak.Square{})
			case "":
			default:
				return nil, fmt.Errorf("bad piece: %v", p)
			}
		}
		if len(row) != len(lines) {
			return nil, errors.New("size mismatch")
		}
		pieces = append(pieces, row)
	}
	ply := 2
	if who == tak.Black {
		ply = 3
	}
	return tak.FromSquares(tak.Config{Size: len(pieces)}, pieces, ply)
}

func TestHardTopCap(t *testing.T) {
	cases := []struct {
		tps string
		v   int64
	}{
		{"x5/x5/x5/x4,2/21C,x3,1 2 3", 0},
		{"x5/x5/x5/x4,2/11C,x3,2 2 3", -1},
		{"x5/x5/x4,2/x4,2/11C,x3,2 1 4", 1},
		{"x5/x5/x4,2/x4,2/21C,x3,1 1 4", 0},
		{"x5/x5/x5/x5/11,x3,22C 1 4", -1},
		{"x5/x5/1,x4/1,x4/1,1,x2,22C 2 4", 1},
	}
	ws := Weights{HardTopCap: 1}
	for _, tc := range cases {
		p, e := ptn.ParseTPS(tc.tps)
		if e != nil {
			t.Fatal("parse ", e)
		}

		c := bitboard.Precompute(uint(p.Size()))

		eval := evaluate(&c, &ws, p)
		if eval != tc.v {
			t.Errorf("%q: got v=%d expect=%v", tc.tps, eval, tc.v)
		}
	}
}

func TestScoreThreats(t *testing.T) {
	ws := Weights{
		Potential: 1,
		Threat:    100,
	}

	cases := []struct {
		board     string
		color     tak.Color
		potential int64
	}{
		{`
. . . . .
. . . . .
. . . . .
. . . . .
. . . . .`, tak.White, 0},
		{`
. . . . .
. . . . .
. . . . .
. . . . .
W . . . B`, tak.White, 0},
		{`
. . . . .
. . . . .
. . . . .
W . . . B
W . . . B`, tak.White, 0},
		{`
. . . . .
W . . B .
W . . . B
W . . . B
W . . . B`, tak.Black, 1},
		{`
. . . . .
W . . B .
W . . . B
W . . . B
W . . . B`, tak.White, 1 << 20},
		{`
BS W . . .
W  . . B .
W  . . . B
W  . . . B
W  . . . B`, tak.Black, 1},
		{`
. W . . .
W . . B .
W . . . B
W . . . B
W . . . B`, tak.Black, 102},
		{`
. W . . .
. W . B .
W . . . B
W . . . B
W . . . B`, tak.Black, 2},
		{`
. W . . .
B W . B .
W B . . B
W . . . B
W . . . B`, tak.Black, 0},
		{`
. W . . .
B W . B .
W B W . B
W . . . B
W . . . B`, tak.Black, 100},
		{`
W W . . B
. W . . B
. W . . B
. . . . .
. . . . B`, tak.White, -1},
		{`
W W . . B
. W . . B
. W . . B
. . . . .
. . . . B`, tak.Black, -(1 << 20)},
		{`
. . B  . .
. . B  . .
. . B  . .
. . .  . .
. . BS . .`, tak.Black, 0,
		},
	}
	for i, tc := range cases {
		pos, e := board(tc.board, tc.color)
		if e != nil {
			t.Errorf("parse %d: %v", i, e)
			continue
		}
		c := bitboard.Precompute(uint(pos.Size()))
		score := scoreThreats(&c, &ws, pos)
		if score != tc.potential {
			t.Errorf("[%d] got potential=%d != %d", i, score, tc.potential)
		}
	}
}

func TestCalculateInfluence(t *testing.T) {
	p, e := board(`
. W . . .
W . W . .
. W . . .
. . . . .
. . . . .
`, tak.White)
	if e != nil {
		t.Fatal(e)
	}
	c := bitboard.Precompute(uint(p.Size()))

	var out [3]uint64
	computeInfluence(&c, p.White, out[:])
	expect := []uint64{
		0x10100,
		0x1405,
		0x40,
	}
	for i, o := range out {
		if o != expect[i] {
			t.Errorf("[%d]=%25s != %25s",
				i,
				strconv.FormatUint(o, 2),
				strconv.FormatUint(expect[i], 2))
		}
	}

	var sat [2]uint64
	computeInfluence(&c, p.White, sat[:])
	if sat[1] != expect[1]|expect[2] {
		t.Error("bad saturate")
	}
}

func TestScoreControl(t *testing.T) {
	ws := Weights{
		EmptyControl: 1,
		FlatControl:  100,
	}
	c := bitboard.Precompute(5)

	cases := []struct {
		board     string
		influence int64
	}{
		{`
. . . . .
. . . . .
. . . . .
. . . . .
. . . . .`, 0},
		{`
. . . . .
. . . . .
. . W . .
. . . . .
. . . . .`, 4},
		{`
. . . . .
. . . . .
. B W B .
. . . . .
. . . . .`, 96},
		{`
. . . . .
. . . . .
. . . . .
. . . . .
W . . . B`, 0},
		{`
. . . . .
. . . . .
. . . . .
W . . . B
W . . . B`, 0},
		{`
. . . . .
W . . B .
W . . . B
W . . . B
W . . . B`, 99},
		{`
. .  .  . .
. .  .  . .
. B  WC . .
. .  B  . .
. .  .  . .`, 197},
	}
	for i, tc := range cases {
		pos, e := board(tc.board, tak.White)
		if e != nil {
			t.Errorf("parse %d: %v", i, e)
			continue
		}
		score := scoreControl(&c, &ws, pos)
		if score != tc.influence {
			t.Errorf("[%d] got influence=%d != %d", i, score, tc.influence)
		}
	}
}
