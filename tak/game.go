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
	p := alloc(&Position{
		cfg:         &g,
		whiteStones: byte(g.Pieces),
		whiteCaps:   byte(g.Capstones),
		blackStones: byte(g.Pieces),
		blackCaps:   byte(g.Capstones),
		move:        0,
	})
	return p
}

type Square []Piece

type Position struct {
	cfg         *Config
	whiteStones byte
	whiteCaps   byte
	blackStones byte
	blackCaps   byte

	move int

	White    uint64
	Black    uint64
	Standing uint64
	Caps     uint64
	Height   []uint8
	Stacks   []uint64

	analysis Analysis
}

type Analysis struct {
	WhiteGroups []uint64
	BlackGroups []uint64
}

// FromSquares initializes a Position with the specified squares and
// move number. `board` is a slice of rows, numbered from low to high,
// each of which is a slice of positions.
func FromSquares(cfg Config, board [][]Square, move int) (*Position, error) {
	p := New(cfg)
	p.move = move
	for y := 0; y < p.Size(); y++ {
		for x := 0; x < p.Size(); x++ {
			sq := board[y][x]
			if len(sq) == 0 {
				continue
			}
			i := uint(x + y*p.Size())
			switch sq[0].Color() {
			case White:
				p.White |= (1 << i)
			case Black:
				p.Black |= (1 << i)
			}
			switch sq[0].Kind() {
			case Capstone:
				p.Caps |= (1 << i)
			case Standing:
				p.Standing |= (1 << i)
			}
			for j, piece := range sq {
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
				if j == 0 {
					continue
				}
				if piece.Color() == Black {
					p.Stacks[i] |= 1 << uint(j-1)
				}
			}
			p.Height[i] = uint8(len(sq))
		}
	}
	p.analyze()
	return p, nil
}

func (p *Position) Size() int {
	return p.cfg.Size
}

func (p *Position) At(x, y int) Square {
	i := uint(x + y*p.Size())
	if (p.White|p.Black)&(1<<i) == 0 {
		return nil
	}
	sq := make(Square, p.Height[i])
	sq[0] = p.Top(x, y)
	for j := uint8(1); j < p.Height[i]; j++ {
		if p.Stacks[i]&(1<<(j-1)) != 0 {
			sq[j] = MakePiece(Black, Flat)
		} else {
			sq[j] = MakePiece(White, Flat)
		}
	}
	return sq
}

func (p *Position) Top(x, y int) Piece {
	i := uint(x + y*p.Size())
	var c Color
	var k Kind
	switch {
	case p.White&(1<<i) != 0:
		c = White
	case p.Black&(1<<i) != 0:
		c = Black
	default:
		return 0
	}
	switch {
	case p.Standing&(1<<i) != 0:
		k = Standing
	case p.Caps&(1<<i) != 0:
		k = Capstone
	default:
		k = Flat
	}
	return MakePiece(c, k)
}

func set(p *Position, x, y int, s Square) {
	i := uint(y*p.cfg.Size + x)
	p.White &= ^(1 << i)
	p.Black &= ^(1 << i)
	p.Standing &= ^(1 << i)
	p.Caps &= ^(1 << i)
	if len(s) == 0 {
		p.Height[i] = 0
		return
	}
	p.Height[i] = uint8(len(s))
	switch s[0].Color() {
	case White:
		p.White |= (1 << i)
	case Black:
		p.Black |= (1 << i)
	}
	switch s[0].Kind() {
	case Standing:
		p.Standing |= (1 << i)
	case Capstone:
		p.Caps |= (1 << i)
	}
	p.Stacks[i] = 0
	for j, piece := range s[1:] {
		if piece.Color() == Black {
			p.Stacks[i] |= (1 << uint(j))
		}
	}
}

func (p *Position) Hash() uint64 {
	return 0
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

	if (p.whiteStones+p.whiteCaps) != 0 &&
		(p.blackStones+p.blackCaps) != 0 &&
		(p.White|p.Black) != p.cfg.c.Mask {
		return false, NoColor
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
	wr := p.White &^ p.Standing
	br := p.Black &^ p.Standing
	alloc := p.analysis.WhiteGroups
	p.analysis.WhiteGroups = bitboard.FloodGroups(&p.cfg.c, wr, alloc)
	alloc = p.analysis.WhiteGroups
	alloc = alloc[len(alloc):len(alloc):cap(alloc)]
	p.analysis.BlackGroups = bitboard.FloodGroups(&p.cfg.c, br, alloc)
}

func (p *Position) countFlats() (w int, b int) {
	w = bitboard.Popcount(p.White &^ (p.Standing | p.Caps))
	b = bitboard.Popcount(p.Black &^ (p.Standing | p.Caps))
	return w, b
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
	Over       bool
	Reason     WinReason
	Winner     Color
	WhiteFlats int
	BlackFlats int
}

func (p *Position) WinDetails() WinDetails {
	over, c := p.GameOver()
	var d WinDetails
	d.Over = over
	d.Winner = c
	d.WhiteFlats, d.BlackFlats = p.countFlats()
	if _, ok := p.hasRoad(); ok {
		d.Reason = RoadWin
	} else {
		d.Reason = FlatsWin
	}
	return d
}
