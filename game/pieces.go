package game

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

func MakePiece(color Color, kind Kind) Piece {
	return Piece(byte(color) | byte(kind))
}

func (p Piece) Color() Color {
	return Color(byte(p) & colorMask)
}

func (p Piece) Kind() Kind {
	return Kind(byte(p) & typeMask)
}

func (p Piece) IsRoad() bool {
	return p.Kind() == Flat || p.Kind() == Capstone
}

func (p Piece) String() string {
	c := ""
	if p.Color() == White {
		c = "W"
	} else {
		c = "B"
	}
	switch p.Kind() {
	case Capstone:
		c += "C"
	case Standing:
		c += "S"
	}
	return c
}

func (c Color) String() string {
	if c == White {
		return "white"
	}
	return "black"
}

func (c Color) Flip() Color {
	if c == White {
		return Black
	}
	return White
}
