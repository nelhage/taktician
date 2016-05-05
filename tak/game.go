package tak

import (
	"errors"

	"github.com/nelhage/taktician/bitboard"
)

type Config struct {
	Size      int
	Pieces    int
	Capstones int

	c bitboard.Constants
}

var defaultPieces = []int{0, 0, 0, 10, 15, 21, 30, 40, 50}
var defaultCaps = []int{0, 0, 0, 0, 0, 1, 1, 1, 2}

func New(g Config) *Position {
	if g.Pieces == 0 {
		g.Pieces = defaultPieces[g.Size]
	}
	if g.Capstones == 0 {
		g.Capstones = defaultCaps[g.Size]
	}
	g.c = bitboard.Precompute(uint(g.Size))
	p := &Position{
		cfg:         &g,
		whiteStones: byte(g.Pieces),
		whiteCaps:   byte(g.Capstones),
		blackStones: byte(g.Pieces),
		blackCaps:   byte(g.Capstones),
		move:        0,
		board:       make([]Square, g.Size*g.Size),
	}
	return p
}

type Square []Piece

type Position struct {
	cfg         *Config
	whiteStones byte
	whiteCaps   byte
	blackStones byte
	blackCaps   byte

	move     int
	board    []Square
	analysis Analysis
}

type Analysis struct {
	WhiteRoad   uint64
	BlackRoad   uint64
	White       uint64
	Black       uint64
	WhiteGroups []uint64
	BlackGroups []uint64
}

// FromSquares initializes a Position with the specified squares and
// move number. `board` is a slice of rows, numbered from low to high,
// each of which is a slice of positions.
func FromSquares(cfg Config, board [][]Square, move int) (*Position, error) {
	p := New(cfg)
	p.move = move
	for x := 0; x < p.Size(); x++ {
		for y := 0; y < p.Size(); y++ {
			p.set(x, y, board[y][x])
			for _, piece := range board[y][x] {
				switch piece {
				case MakePiece(White, Capstone):
					p.whiteCaps--
				case MakePiece(Black, Capstone):
					p.blackCaps--
				case MakePiece(White, Flat), MakePiece(White, Standing):
					p.whiteStones--
				case MakePiece(Black, Flat), MakePiece(Black, Standing):
					p.blackStones--
				default:
					return nil, errors.New("bad stone")
				}
			}
		}
	}
	p.analyze()
	return p, nil
}

func (p *Position) Size() int {
	return p.cfg.Size
}

func (p *Position) At(x, y int) Square {
	return p.board[y*p.cfg.Size+x]
}

func (p *Position) set(x, y int, s Square) {
	p.board[y*p.cfg.Size+x] = s
}

func (p *Position) ToMove() Color {
	if p.move%2 == 0 {
		return White
	}
	return Black
}

func (p *Position) MoveNumber() int {
	return p.move
}

func (p *Position) WhiteStones() int {
	return int(p.whiteStones)
}

func (p *Position) BlackStones() int {
	return int(p.blackStones)
}

func (p *Position) GameOver() (over bool, winner Color) {
	if p, ok := p.hasRoad(); ok {
		return true, p
	}

	if (p.whiteStones+p.whiteCaps) != 0 && (p.blackStones+p.blackCaps) != 0 {
		return false, White
	}

	return true, p.flatsWinner()
}

func (p *Position) roadAt(x, y int) (Color, bool) {
	sq := p.At(x, y)
	if len(sq) == 0 {
		return White, false
	}
	return sq[0].Color(), sq[0].IsRoad()
}

func (p *Position) hasRoad() (Color, bool) {
	white, black := false, false

	for _, g := range p.analysis.WhiteGroups {
		if ((g&p.cfg.c.T) != 0 && (g&p.cfg.c.B) != 0) ||
			((g&p.cfg.c.L) != 0 && (g&p.cfg.c.R) != 0) {
			white = true
			break
		}
	}
	for _, g := range p.analysis.BlackGroups {
		if ((g&p.cfg.c.T) != 0 && (g&p.cfg.c.B) != 0) ||
			((g&p.cfg.c.L) != 0 && (g&p.cfg.c.R) != 0) {
			black = true
			break
		}
	}

	switch {
	case white && black:
		if p.ToMove() == White {
			return Black, true
		}
		return White, true
	case white:
		return White, true
	case black:
		return Black, true
	default:
		return White, false
	}

}

func (p *Position) Analysis() *Analysis {
	return &p.analysis
}

func (p *Position) analyze() {
	var br uint64
	var wr uint64
	var b uint64
	var w uint64
	for i, sq := range p.board {
		if len(sq) == 0 {
			continue
		}
		if sq[0].Color() == White {
			w |= 1 << uint(i)
		} else {
			b |= 1 << uint(i)
		}
		if sq[0].IsRoad() {
			if sq[0].Color() == White {
				wr |= 1 << uint(i)
			} else {
				br |= 1 << uint(i)
			}
		}
	}
	p.analysis.WhiteRoad = wr
	p.analysis.BlackRoad = br
	p.analysis.White = w
	p.analysis.Black = b

	alloc := make([]uint64, 0, 2*p.Size())
	p.analysis.WhiteGroups = p.floodone(wr, alloc)
	alloc = p.analysis.WhiteGroups
	p.analysis.BlackGroups = p.floodone(br, alloc[len(alloc):len(alloc):cap(alloc)])
}

func (p *Position) floodone(bits uint64, out []uint64) []uint64 {
	var seen uint64
	for bits != 0 {
		next := bits & (bits - 1)
		bit := bits &^ next

		if seen&bit == 0 {
			g := bitboard.Flood(&p.cfg.c, bits, bit)
			if g != bit && bitboard.Popcount(g) > 2 {
				out = append(out, g)
			}
			seen |= g
		}

		bits = next
	}
	return out
}

func (p *Position) bitroad(bits uint64) bool {
	s := uint(p.cfg.Size)
	var mask uint64 = (1 << s) - 1
	row := bits & mask
	for i := uint(1); i < s; i++ {
		if row == 0 {
			return false
		}
		next := (bits >> (i * s)) & mask
		row &= next
		for {
			last := row
			row |= ((row >> 1) & next) |
				((row << 1) & next)
			row &= mask
			if row == last {
				break
			}
		}
	}
	return row != 0

}

func (p *Position) countFlats() (w int, b int) {
	cw, cb := 0, 0
	for i := 0; i < p.cfg.Size*p.cfg.Size; i++ {
		stack := p.board[i]
		if len(stack) > 0 {
			if stack[0].Kind() == Flat {
				if stack[0].Color() == White {
					cw++
				} else {
					cb++
				}
			}
		}
	}
	return cw, cb
}

func (p *Position) flatsWinner() Color {
	cw, cb := p.countFlats()
	if cw > cb {
		return White
	}
	if cb > cw {
		return Black
	}
	return NoColor
}

type WinReason int

const (
	RoadWin WinReason = iota
	FlatsWin
	Resignation
)

type WinDetails struct {
	Reason     WinReason
	Winner     Color
	WhiteFlats int
	BlackFlats int
}

func (p *Position) WinDetails() WinDetails {
	over, c := p.GameOver()
	if !over {
		panic("WinDetails on a game not over")
	}
	var d WinDetails
	d.Winner = c
	d.WhiteFlats, d.BlackFlats = p.countFlats()
	if _, ok := p.hasRoad(); ok {
		d.Reason = RoadWin
	} else {
		d.Reason = FlatsWin
	}
	return d
}
