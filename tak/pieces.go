package tak

import "fmt"

type Color byte
type Kind byte
type Piece byte

const (
	White   Color = 1 << 7
	Black   Color = 1 << 6
	NoColor Color = 0

	colorMask byte = 3 << 6

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
	switch c {
	case White:
		return "white"
	case Black:
		return "black"
	case NoColor:
		return "no color"
	default:
		panic(fmt.Sprintf("bad color: %x", int(c)))
	}
}

func (c Color) Flip() Color {
	switch c {
	case White:
		return Black
	case Black:
		return White
	case NoColor:
		return NoColor
	default:
		panic(fmt.Sprintf("bad color: %x", int(c)))
	}
}
