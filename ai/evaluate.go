package ai

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/nelhage/taktician/bitboard"
	"github.com/nelhage/taktician/tak"
)

const (
	endgameCutoff = 7
)

type FlatScores struct {
	Hard, Soft int
}

type Weights struct {
	TopFlat     int
	EndgameFlat int
	Standing    int
	Capstone    int

	FlatCaptives     FlatScores
	StandingCaptives FlatScores
	CapstoneCaptives FlatScores

	Liberties      int
	GroupLiberties int

	Groups [8]int
}

var defaultWeights = Weights{
	TopFlat:     400,
	EndgameFlat: 800,
	Standing:    200,
	Capstone:    300,

	FlatCaptives: FlatScores{
		Hard: 125,
		Soft: -75,
	},
	StandingCaptives: FlatScores{
		Hard: 125,
		Soft: -50,
	},
	CapstoneCaptives: FlatScores{
		Hard: 150,
		Soft: -25,
	},

	GroupLiberties: 20,

	Groups: [8]int{
		0,   // 0
		0,   // 1
		0,   // 2
		100, // 3
		300, // 4
	},
}

var defaultWeights6 = Weights{
	TopFlat:     400,
	EndgameFlat: 800,
	Standing:    200,
	Capstone:    300,

	FlatCaptives: FlatScores{
		Hard: 125,
		Soft: -200,
	},
	StandingCaptives: FlatScores{
		Hard: 125,
		Soft: -150,
	},
	CapstoneCaptives: FlatScores{
		Hard: 150,
		Soft: -50,
	},

	GroupLiberties: 20,

	Groups: [8]int{
		0,   // 0
		0,   // 1
		0,   // 2
		100, // 3
		300, // 4
		500, // 5
	},
}

var DefaultWeights = []Weights{
	defaultWeights,  // 0
	defaultWeights,  // 1
	defaultWeights,  // 2
	defaultWeights,  // 3
	defaultWeights,  // 4
	defaultWeights,  // 5
	defaultWeights6, // 6
	defaultWeights,  // 7
	defaultWeights,  // 8
}

func MakeEvaluator(size int, w *Weights) EvaluationFunc {
	if w == nil {
		w = &DefaultWeights[size]
	}
	return func(c *bitboard.Constants, p *tak.Position) int64 {
		return evaluate(c, w, p)
	}
}

func evaluateTerminal(p *tak.Position, winner tak.Color) int64 {
	var pieces int64
	if winner == tak.White {
		pieces = int64(p.WhiteStones())
	} else {
		pieces = int64(p.BlackStones())
	}
	switch winner {
	case tak.NoColor:
		return 0
	case p.ToMove():
		return MaxEval - int64(p.MoveNumber()) + pieces
	default:
		return MinEval + int64(p.MoveNumber()) - pieces
	}
}

func EvaluateWinner(_ *bitboard.Constants, p *tak.Position) int64 {
	if over, winner := p.GameOver(); over {
		return evaluateTerminal(p, winner)
	}
	return 0
}

func evaluate(c *bitboard.Constants, w *Weights, p *tak.Position) int64 {
	if over, winner := p.GameOver(); over {
		return evaluateTerminal(p, winner)
	}

	var ws, bs int64

	analysis := p.Analysis()

	left := p.WhiteStones()
	if p.BlackStones() < left {
		left = p.BlackStones()
	}
	if left > endgameCutoff {
		left = endgameCutoff
	}
	flat := w.TopFlat + ((endgameCutoff-left)*w.EndgameFlat)/endgameCutoff
	if p.ToMove() == tak.White {
		ws += int64(flat/2) + 50
	} else {
		bs += int64(flat/2) + 50
	}

	ws += int64(bitboard.Popcount(p.White&^(p.Caps|p.Standing)) * flat)
	bs += int64(bitboard.Popcount(p.Black&^(p.Caps|p.Standing)) * flat)
	ws += int64(bitboard.Popcount(p.White&p.Standing) * w.Standing)
	bs += int64(bitboard.Popcount(p.Black&p.Standing) * w.Standing)
	ws += int64(bitboard.Popcount(p.White&p.Caps) * w.Capstone)
	bs += int64(bitboard.Popcount(p.Black&p.Caps) * w.Capstone)

	for i, h := range p.Height {
		if h <= 1 {
			continue
		}
		bit := uint64(1 << uint(i))
		s := p.Stacks[i] & ((1 << (h - 1)) - 1)
		var hf, sf int
		var ptr *int64
		if p.White&bit != 0 {
			sf = bitboard.Popcount(s)
			hf = int(h) - sf - 1
			ptr = &ws
		} else {
			hf = bitboard.Popcount(s)
			sf = int(h) - hf - 1
			ptr = &bs
		}

		switch {
		case p.Standing&(1<<uint(i)) != 0:
			*ptr += (int64(hf*w.StandingCaptives.Hard) +
				int64(sf*w.StandingCaptives.Soft))
		case p.Caps&(1<<uint(i)) != 0:
			*ptr += (int64(hf*w.CapstoneCaptives.Hard) +
				int64(sf*w.CapstoneCaptives.Soft))
		default:
			*ptr += (int64(hf*w.FlatCaptives.Hard) +
				int64(sf*w.FlatCaptives.Soft))
		}
	}

	ws += int64(scoreGroups(c, analysis.WhiteGroups, w, p.Black|p.Standing))
	bs += int64(scoreGroups(c, analysis.BlackGroups, w, p.White|p.Standing))

	if w.Liberties != 0 {
		wr := p.White &^ p.Standing
		br := p.Black &^ p.Standing
		wl := bitboard.Popcount(bitboard.Grow(c, ^p.Black, wr) &^ p.White)
		bl := bitboard.Popcount(bitboard.Grow(c, ^p.White, br) &^ p.Black)
		ws += int64(w.Liberties * wl)
		bs += int64(w.Liberties * bl)
	}

	if p.ToMove() == tak.White {
		return ws - bs
	}
	return bs - ws
}

func scoreGroups(c *bitboard.Constants, gs []uint64, ws *Weights, other uint64) int {
	sc := 0
	var allg uint64
	for _, g := range gs {
		w, h := bitboard.Dimensions(c, g)

		sc += ws.Groups[w]
		sc += ws.Groups[h]
		allg |= g
	}
	if ws.GroupLiberties != 0 {
		libs := bitboard.Popcount(bitboard.Grow(c, ^other, allg) &^ allg)
		sc += libs * ws.GroupLiberties
	}

	return sc
}

func ExplainScore(m *MinimaxAI, out io.Writer, p *tak.Position) {
	tw := tabwriter.NewWriter(out, 4, 8, 1, '\t', 0)
	fmt.Fprintf(tw, "\twhite\tblack\n")
	var scores [2]struct {
		flats    int
		standing int
		caps     int

		stones   int
		captured int
	}

	scores[0].flats = bitboard.Popcount(p.White &^ (p.Caps | p.Standing))
	scores[1].flats = bitboard.Popcount(p.Black &^ (p.Caps | p.Standing))
	scores[0].standing = bitboard.Popcount(p.White & p.Standing)
	scores[1].standing = bitboard.Popcount(p.Black & p.Standing)
	scores[0].caps = bitboard.Popcount(p.White & p.Caps)
	scores[1].caps = bitboard.Popcount(p.Black & p.Caps)

	for i, h := range p.Height {
		if h <= 1 {
			continue
		}
		s := p.Stacks[i] & ((1 << (h - 1)) - 1)
		bf := bitboard.Popcount(s)
		wf := int(h) - bf - 1
		scores[0].stones += wf
		scores[1].stones += bf

		captured := int(h - 1)
		if captured > p.Size()-1 {
			captured = p.Size() - 1
		}
		if p.White&(1<<uint(i)) != 0 {
			scores[0].captured += captured
		} else {
			scores[1].captured += captured
		}
	}

	fmt.Fprintf(tw, "flats\t%d\t%d\n", scores[0].flats, scores[1].flats)
	fmt.Fprintf(tw, "standing\t%d\t%d\n", scores[0].standing, scores[1].standing)
	fmt.Fprintf(tw, "caps\t%d\t%d\n", scores[0].caps, scores[1].caps)
	fmt.Fprintf(tw, "captured\t%d\t%d\n", scores[0].captured, scores[1].captured)
	fmt.Fprintf(tw, "stones\t%d\t%d\n", scores[0].stones, scores[1].stones)

	analysis := p.Analysis()

	wr := p.White &^ p.Standing
	br := p.Black &^ p.Standing
	wl := bitboard.Popcount(bitboard.Grow(&m.c, ^p.Black, wr) &^ p.White)
	bl := bitboard.Popcount(bitboard.Grow(&m.c, ^p.White, br) &^ p.Black)

	fmt.Fprintf(tw, "liberties\t%d\t%d\n", wl, bl)

	var allg uint64
	for i, g := range analysis.WhiteGroups {
		w, h := bitboard.Dimensions(&m.c, g)
		fmt.Fprintf(tw, "g%d\t%dx%x\n", i, w, h)
		allg |= g
	}
	wgl := bitboard.Popcount(bitboard.Grow(&m.c, m.c.Mask&^(p.Black|p.Standing), allg) &^ allg)
	allg = 0
	for i, g := range analysis.BlackGroups {
		w, h := bitboard.Dimensions(&m.c, g)
		fmt.Fprintf(tw, "g%d\t\t%dx%x\n", i, w, h)
		allg |= g
	}
	bgl := bitboard.Popcount(bitboard.Grow(&m.c, m.c.Mask&^(p.White|p.Standing), allg) &^ allg)
	fmt.Fprintf(tw, "gl\t%d\t%d\n", wgl, bgl)
	tw.Flush()
}
