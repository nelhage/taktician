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

	Potential int
	Threat    int

	Influence int
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
		Soft: -50,
	},

	Liberties: 20,

	Groups: [8]int{
		0,   // 0
		0,   // 1
		0,   // 2
		100, // 3
		300, // 4
	},

	Potential: 100,
	Threat:    300,
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
		Soft: -75,
	},

	Liberties: 20,

	Groups: [8]int{
		0,   // 0
		0,   // 1
		0,   // 2
		100, // 3
		300, // 4
		500, // 5
	},

	Potential: 100,
	Threat:    300,
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

const moveScale = 100

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
		return MaxEval - moveScale*int64(p.MoveNumber()) + pieces
	default:
		return MinEval + moveScale*int64(p.MoveNumber()) - pieces
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

	var score int64

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
		score += int64(flat/2) + 50
	} else {
		score -= int64(flat/2) + 50
	}

	score += int64(bitboard.Popcount(p.White&^(p.Caps|p.Standing)) * flat)
	score -= int64(bitboard.Popcount(p.Black&^(p.Caps|p.Standing)) * flat)
	score += int64(bitboard.Popcount(p.White&p.Standing) * w.Standing)
	score -= int64(bitboard.Popcount(p.Black&p.Standing) * w.Standing)
	score += int64(bitboard.Popcount(p.White&p.Caps) * w.Capstone)
	score -= int64(bitboard.Popcount(p.Black&p.Caps) * w.Capstone)

	for i, h := range p.Height {
		if h <= 1 {
			continue
		}
		bit := uint64(1 << uint(i))
		s := p.Stacks[i] & ((1 << (h - 1)) - 1)
		var hf, sf int
		var sign int64
		if p.White&bit != 0 {
			sf = bitboard.Popcount(s)
			hf = int(h) - sf - 1
			sign = 1
		} else {
			hf = bitboard.Popcount(s)
			sf = int(h) - hf - 1
			sign = -1
		}

		switch {
		case p.Standing&(1<<uint(i)) != 0:
			score += sign * (int64(hf*w.StandingCaptives.Hard) +
				int64(sf*w.StandingCaptives.Soft))
		case p.Caps&(1<<uint(i)) != 0:
			score += sign * (int64(hf*w.CapstoneCaptives.Hard) +
				int64(sf*w.CapstoneCaptives.Soft))
		default:
			score += sign * (int64(hf*w.FlatCaptives.Hard) +
				int64(sf*w.FlatCaptives.Soft))
		}
	}

	score += int64(scoreGroups(c, analysis.WhiteGroups, w, p.Black|p.Standing))
	score -= int64(scoreGroups(c, analysis.BlackGroups, w, p.White|p.Standing))

	if w.Liberties != 0 {
		wr := p.White &^ p.Standing
		br := p.Black &^ p.Standing
		wl := bitboard.Popcount(bitboard.Grow(c, ^p.Black, wr) &^ p.White)
		bl := bitboard.Popcount(bitboard.Grow(c, ^p.White, br) &^ p.Black)
		score += int64(w.Liberties * wl)
		score -= int64(w.Liberties * bl)
	}

	score += scoreThreats(c, w, p)
	score += scoreInfluence(c, w, p)

	if p.ToMove() == tak.White {
		return score
	}
	return -score
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

func scoreThreats(c *bitboard.Constants, ws *Weights, p *tak.Position) int64 {
	if ws.Potential == 0 && ws.Threat == 0 {
		return 0
	}
	analysis := p.Analysis()
	empty := c.Mask &^ (p.White | p.Black)

	countOne := func(gs []uint64, pieces uint64) (int, int) {
		var place, threat int
		singles := pieces
		for _, g := range gs {
			singles &= ^g
		}
		for i, g := range gs {
			if g&c.Edge == 0 {
				continue
			}
			slides := bitboard.Grow(c, c.Mask&^(p.Standing|p.Caps), pieces&^g)
			var pmap, tmap uint64
			if g&c.L != 0 {
				pmap |= (g >> 1) & empty & c.R
				tmap |= (g >> 1) & slides & c.R
			}
			if g&c.R != 0 {
				pmap |= (g << 1) & empty & c.L
				tmap |= (g << 1) & slides & c.L
			}
			if g&c.T != 0 {
				pmap |= (g >> c.Size) & empty & c.B
				tmap |= (g >> c.Size) & slides & c.B
			}
			if g&c.B != 0 {
				pmap |= (g << c.Size) & empty & c.T
				tmap |= (g << c.Size) & slides & c.T
			}
			s := singles
			j := 0
			for {
				var other uint64
				if j < i {
					other = gs[j]
					j++
				} else if s != 0 {
					next := s & (s - 1)
					other = s &^ next
					s = next
				} else {
					break
				}
				if !((g&c.L != 0 && other&c.R != 0) ||
					(g&c.R != 0 && other&c.L != 0) ||
					(g&c.B != 0 && other&c.T != 0) ||
					(g&c.T != 0 && other&c.B != 0)) {
					continue
				}
				slides := bitboard.Grow(c, c.Mask&^(p.Standing|p.Caps), pieces&^(g|other))
				isect := bitboard.Grow(c, c.Mask, g) &
					bitboard.Grow(c, c.Mask, other)
				pmap |= isect & empty
				tmap |= isect & slides
			}
			place += bitboard.Popcount(pmap)
			threat += bitboard.Popcount(tmap)
		}
		return place, threat
	}
	wp, wt := countOne(analysis.WhiteGroups, p.White)
	bp, bt := countOne(analysis.BlackGroups, p.Black)

	if wp+wt > 0 && p.ToMove() == tak.White {
		return 1 << 20
	}
	if bp+bt > 0 && p.ToMove() == tak.White {
		return -(1 << 20)
	}

	return int64((wp-bp)*ws.Potential) + int64((wt-bt)*ws.Threat)
}

func computeInfluence(c *bitboard.Constants, mine uint64, out []uint64) {
	for mine != 0 {
		next := mine & (mine - 1)
		bit := mine &^ next
		mine = next

		g := bitboard.Grow(c, c.Mask, bit) &^ bit

		carry := g
		for i := 0; carry != 0 && i < len(out); i++ {
			cout := out[i] & carry
			out[i] ^= carry
			carry = cout
		}
		if carry != 0 {
			out[len(out)-1] |= carry
		}
	}
}

func scoreInfluence(c *bitboard.Constants, ws *Weights, p *tak.Position) int64 {
	if ws.Influence == 0 {
		return 0
	}
	var wi, bi [3]uint64
	computeInfluence(c, p.White&^(p.Caps|p.Standing), wi[:])
	computeInfluence(c, p.Black&^(p.Caps|p.Standing), bi[:])
	var bc, wc uint64
	for i := len(wi) - 1; i >= 0; i-- {
		wb := wi[i] &^ (wc | bc)
		bb := bi[i] &^ (wc | bc)

		wc |= (wb &^ bb)
		bc |= (bb &^ wb)
	}
	return int64(ws.Influence * (bitboard.Popcount(wc) - bitboard.Popcount(bc)))
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
