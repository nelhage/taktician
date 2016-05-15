package ai

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/nelhage/taktician/bitboard"
	"github.com/nelhage/taktician/tak"
)

type Weights struct {
	TopFlat  int
	Standing int
	Capstone int

	Flat     int
	Captured int

	Liberties int

	Tempo int

	Groups [8]int
}

var DefaultWeights = Weights{
	TopFlat:  300,
	Standing: 200,
	Capstone: 300,

	Flat:      100,
	Liberties: 25,

	Captured: 25,

	Tempo: 250,

	Groups: [8]int{
		0,   // 0
		0,   // 1
		0,   // 2
		100, // 3
		300, // 4
		500, // 5
	},
}

func MakeEvaluator(w *Weights) EvaluationFunc {
	return func(m *MinimaxAI, p *tak.Position) int64 {
		return evaluate(w, m, p)
	}
}

var DefaultEvaluate = MakeEvaluator(&DefaultWeights)

func evaluate(w *Weights, m *MinimaxAI, p *tak.Position) int64 {
	if over, winner := p.GameOver(); over {
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
			return maxEval - int64(p.MoveNumber()) + pieces
		default:
			return minEval + int64(p.MoveNumber()) - pieces
		}
	}
	mine, theirs := 0, 0
	me := p.ToMove()
	addw := func(c tak.Color, w int) {
		if c == me {
			mine += w
		} else {
			theirs += w
		}
	}
	addw(p.ToMove(), w.Tempo)
	analysis := p.Analysis()
	for x := 0; x < p.Size(); x++ {
		for y := 0; y < p.Size(); y++ {
			sq := p.At(x, y)
			if len(sq) == 0 {
				continue
			}
			switch sq[0].Kind() {
			case tak.Standing:
				addw(sq[0].Color(), w.Standing)
			case tak.Flat:
				addw(sq[0].Color(), w.TopFlat)
			case tak.Capstone:
				addw(sq[0].Color(), w.Capstone)
			}

			for i, stone := range sq {
				if i > 0 && i < p.Size() {
					addw(sq[0].Color(), w.Captured)
				}
				if stone.Kind() == tak.Flat {
					addw(stone.Color(), w.Flat)
				}
			}
		}
	}

	addw(tak.White, m.scoreGroups(analysis.WhiteGroups, w))
	addw(tak.Black, m.scoreGroups(analysis.BlackGroups, w))

	wl := bitboard.Popcount(bitboard.Grow(&m.c, ^analysis.Black, analysis.WhiteRoad) &^ analysis.WhiteRoad)
	bl := bitboard.Popcount(bitboard.Grow(&m.c, ^analysis.White, analysis.BlackRoad) &^ analysis.BlackRoad)
	addw(tak.White, w.Liberties*wl)
	addw(tak.Black, w.Liberties*bl)

	return int64(mine - theirs)
}

func (ai *MinimaxAI) scoreGroups(gs []uint64, ws *Weights) int {
	sc := 0
	for _, g := range gs {
		w, h := bitboard.Dimensions(&ai.c, g)

		sc += ws.Groups[w]
		sc += ws.Groups[h]
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
	for x := 0; x < p.Size(); x++ {
		for y := 0; y < p.Size(); y++ {
			sq := p.At(x, y)
			if len(sq) == 0 {
				continue
			}
			switch sq[0].Kind() {
			case tak.Standing:
				if sq[0].Color() == tak.White {
					scores[0].standing++
				} else {
					scores[1].standing++
				}
			case tak.Flat:
				if sq[0].Color() == tak.White {
					scores[0].flats++
				} else {
					scores[1].flats++
				}
			case tak.Capstone:
				if sq[0].Color() == tak.White {
					scores[0].caps++
				} else {
					scores[1].caps++
				}
			}

			for i, stone := range sq {
				if i > 0 && i < p.Size() {
					if sq[0].Color() == tak.White {
						scores[0].captured++
					} else {
						scores[1].captured++
					}
				}
				if stone.Kind() == tak.Flat {
					if sq[0].Color() == tak.White {
						scores[0].stones++
					} else {
						scores[1].stones++
					}
				}
			}
		}
	}
	fmt.Fprintf(tw, "flats\t%d\t%d\n", scores[0].flats, scores[1].flats)
	fmt.Fprintf(tw, "standing\t%d\t%d\n", scores[0].standing, scores[1].standing)
	fmt.Fprintf(tw, "caps\t%d\t%d\n", scores[0].caps, scores[1].caps)
	fmt.Fprintf(tw, "capured\t%d\t%d\n", scores[0].captured, scores[1].captured)
	fmt.Fprintf(tw, "stones\t%d\t%d\n", scores[0].stones, scores[1].stones)

	analysis := p.Analysis()

	wl := bitboard.Popcount(bitboard.Grow(&m.c, ^analysis.Black, analysis.WhiteRoad) &^ analysis.WhiteRoad)
	bl := bitboard.Popcount(bitboard.Grow(&m.c, ^analysis.White, analysis.BlackRoad) &^ analysis.BlackRoad)

	fmt.Fprintf(tw, "liberties\t%d\t%d\n", wl, bl)

	for i, g := range analysis.WhiteGroups {
		w, h := bitboard.Dimensions(&m.c, g)
		fmt.Fprintf(tw, "g%d\t%dx%x\n", i, w, h)
	}
	for i, g := range analysis.BlackGroups {
		w, h := bitboard.Dimensions(&m.c, g)
		fmt.Fprintf(tw, "g%d\t\t%dx%x\n", i, w, h)
	}
	tw.Flush()
}
