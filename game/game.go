package game

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

type Color byte
type Kind byte
type Piece byte

const (
	White Color = 1 << 7
	Black Color = 0 << 7

	colorMask byte = 1 << 7

	Flat     Kind = 1
	Standing Kind = 2
	Capstone Kind = 3

	typeMask byte = 1<<2 - 1
)

func makePiece(color Color, kind Kind) Piece {
	return Piece(byte(color) | byte(kind))
}

func pieceColor(p Piece) Color {
	return Color(byte(p) & colorMask)
}

func pieceKind(p Piece) Kind {
	return Kind(byte(p) & typeMask)
}

func (p Piece) String() string {
	c := ""
	if pieceColor(p) == White {
		c = "W"
	} else {
		c = "B"
	}
	switch pieceKind(p) {
	case Capstone:
		c += "C"
	case Standing:
		c += "S"
	}
	return c
}

func flip(c Color) Color {
	if c == White {
		return Black
	}
	return White
}

func isRoad(p Piece) bool {
	return pieceKind(p) == Flat || pieceKind(p) == Capstone
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
	return pieceColor(sq[0]), isRoad(sq[0])
}

func (p *Position) hasRoad() (Color, bool) {
	s := p.cfg.Size
	white, black := false, false
	reachable := make([]Piece, s*s)
	for x := 0; x < s; x++ {
		if c, ok := p.roadAt(x, 0); ok {
			reachable[x] = makePiece(c, Flat)
		}
	}
	for y := 1; y < s; y++ {
		for x := 0; x < s; x++ {
			c, ok := p.roadAt(x, y)
			if !ok {
				continue
			}
			if reachable[x+(y-1)*s] == makePiece(c, Flat) {
				reachable[x+y*s] = makePiece(c, Flat)
			}
		}
		for x := 0; x < s; x++ {
			c, ok := p.roadAt(x, y)
			if !ok {
				continue
			}
			if x > 0 && reachable[x-1+y*s] == makePiece(c, Flat) {
				reachable[x+y*s] = makePiece(c, Flat)
			}
			if x < s-1 && reachable[x+1+y*s] == makePiece(c, Flat) {
				reachable[x+y*s] = makePiece(c, Flat)
			}
		}
	}

	for x := 0; x < s; x++ {
		r := reachable[x+(s-1)*s]
		if r == makePiece(White, Flat) {
			white = true
		}
		if r == makePiece(Black, Flat) {
			black = true
		}
	}

	for i := range reachable {
		reachable[i] = Piece(0)
	}
	for y := 0; y < s; y++ {
		if c, ok := p.roadAt(0, y); ok {
			reachable[y*s] = makePiece(c, Flat)
		}
	}
	for x := 1; x < s; x++ {
		for y := 0; y < s; y++ {
			c, ok := p.roadAt(x, y)
			if !ok {
				continue
			}
			if reachable[x-1+y*s] == makePiece(c, Flat) {
				reachable[x+y*s] = makePiece(c, Flat)
			}
		}
		for y := 0; y < s; y++ {
			c, ok := p.roadAt(x, y)
			if !ok {
				continue
			}
			if y > 0 && reachable[x+(y-1)*s] == makePiece(c, Flat) {
				reachable[x+y*s] = makePiece(c, Flat)
			}
			if y < s-1 && reachable[x+(y+1)*s] == makePiece(c, Flat) {
				reachable[x+y*s] = makePiece(c, Flat)
			}
		}
	}
	for y := 0; y < s; y++ {
		r := reachable[y*s+s-1]
		if r == makePiece(White, Flat) {
			white = true
		}
		if r == makePiece(Black, Flat) {
			black = true
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

func (p *Position) flatsWinner() Color {
	cw, cb := 0, 0
	for i := 0; i < p.cfg.Size*p.cfg.Size; i++ {
		stack := p.board[i]
		if len(stack) > 0 {
			if pieceKind(stack[0]) == Flat {
				if pieceColor(stack[0]) == White {
					cw++
				} else {
					cb++
				}
			}
		}
	}
	if cw > cb {
		return White
	}
	return Black
}
