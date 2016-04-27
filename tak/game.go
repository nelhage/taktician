package tak

import "errors"

type Config struct {
	Size      int
	Pieces    int
	Capstones int
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

	move  int
	board []Square
}

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

	if p.whiteStones != 0 && p.blackStones != 0 {
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
	s := p.cfg.Size
	white, black := false, false
	var rwx uint64
	var rbx uint64
	var rwy uint64
	var rby uint64

	for x := 0; x < s; x++ {
		for y := 0; y < s; y++ {
			if c, ok := p.roadAt(x, y); ok {
				by := uint64(1 << uint(y*s+x))
				bx := uint64(1 << uint(x*s+y))
				if c == White {
					rwy |= by
					rwx |= bx
				} else {
					rby |= by
					rbx |= bx
				}
			}
		}
	}
	white = p.bitroad(rwx) || p.bitroad(rwy)
	black = p.bitroad(rbx) || p.bitroad(rby)

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
		row |= ((row >> 1) & next) |
			((row << 1) & next)
		row &= mask
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
