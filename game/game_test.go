package game

import (
	"reflect"
	"testing"
)

func TestHasRoad(t *testing.T) {
	g := &Config{Size: 5}
	p := &Position{
		cfg:         g,
		whiteStones: 5,
		blackStones: 5,
		move:        2,
		board:       make([]Square, 5*5),
	}

	_, ok := p.hasRoad()
	if ok {
		t.Errorf("empty board hasRoad!")
	}

	for y := 0; y < 5; y++ {
		p.board[y*5+2] = []Piece{makePiece(Black, Flat)}
	}
	c, ok := p.hasRoad()
	if !ok || c != Black {
		t.Errorf("c=%v hasRoad=%v\n", c, ok)
	}

	p.board[0*5+2] = nil
	p.board[0*5+1] = []Piece{makePiece(Black, Flat)}
	p.board[1*5+1] = []Piece{makePiece(Black, Flat)}
	c, ok = p.hasRoad()
	if !ok || c != Black {
		t.Errorf("c=%v hasRoad=%v\n", c, ok)
	}

	p.board[1*5+1] = []Piece{makePiece(Black, Standing)}
	c, ok = p.hasRoad()
	if ok {
		t.Errorf("c=%v hasRoad=%v\n", c, ok)
	}

	p.board = make([]Square, 5*5)
	p.board[1*5+0] = []Piece{makePiece(White, Flat)}
	p.board[1*5+1] = []Piece{makePiece(White, Flat)}
	p.board[2*5+1] = []Piece{makePiece(White, Flat)}
	p.board[2*5+2] = []Piece{makePiece(White, Flat)}
	p.board[3*5+2] = []Piece{makePiece(White, Flat)}
	p.board[3*5+3] = []Piece{makePiece(White, Flat)}
	p.board[4*5+3] = []Piece{makePiece(White, Flat)}
	p.board[4*5+4] = []Piece{makePiece(White, Flat)}

	c, ok = p.hasRoad()
	if !ok || c != White {
		t.Errorf("c=%v hasRoad=%v\n", c, ok)
	}
}

func TestMove(t *testing.T) {
	g := &Config{Size: 5}
	p := &Position{
		cfg:         g,
		whiteStones: 5,
		whiteCaps:   1,
		blackStones: 5,
		blackCaps:   1,
		move:        2,
		board:       make([]Square, 5*5),
	}

	t.Log("Place a flat stone")
	n, e := p.Move(Move{3, 3, PlaceFlat, nil})
	if e != nil {
		t.Fatalf("place: %v", e)
	}
	if sq := n.At(3, 3); len(sq) != 1 || sq[0] != makePiece(White, Flat) {
		t.Fatalf("place failed: %v", sq)
	}
	if sq := p.At(3, 3); len(sq) != 0 {
		t.Fatalf("move mutated original")
	}
	if n.move != 3 {
		t.Fatalf("increment move: %v", n.move)
	}
	if n.whiteStones != 4 {
		t.Fatalf("did not decrement white: %v", n.whiteStones)
	}

	t.Log("Place a standing stone")
	n, e = n.Move(Move{3, 4, PlaceStanding, nil})
	if e != nil {
		t.Fatalf("move 2: %v", e)
	}
	if sq := n.At(3, 4); len(sq) != 1 || sq[0] != makePiece(Black, Standing) {
		t.Fatalf("place failed: %v", sq)
	}

	t.Log("Slide onto a standing")
	_, e = n.Move(Move{3, 3, SlideDown, []byte{1}})
	if e != ErrIllegalSlide {
		t.Fatalf("slide onto wall allowed: %v", e)
	}

	t.Log("Slide onto an empty square")
	nn, e := n.Move(Move{3, 3, SlideUp, []byte{1}})
	if e != nil {
		t.Fatalf("slide up: %v", e)
	}
	if sq := nn.At(3, 3); len(sq) != 0 {
		t.Fatalf("slide did not clear src: %v", sq)
	}
	if sq := nn.At(3, 2); len(sq) != 1 || sq[0] != makePiece(White, Flat) {
		t.Fatalf("slide did not move: %v", sq)
	}
	if sq := n.At(3, 3); len(sq) != 1 || sq[0] != makePiece(White, Flat) {
		t.Fatalf("slide mutated src")
	}
	if sq := n.At(3, 2); len(sq) != 0 {
		t.Fatalf("slide mutated dest in orig")
	}

	t.Log("Place a capstone")
	n, e = nn.Move(Move{3, 3, PlaceCapstone, nil})
	if e != nil {
		t.Fatalf("place cap: %v", e)
	}
	if sq := n.At(3, 3); len(sq) != 1 || sq[0] != makePiece(Black, Capstone) {
		t.Fatalf("place failed: %v", sq)
	}
	if n.blackStones != 4 {
		t.Fatalf("black stones: %d", n.blackStones)
	}
	if n.blackCaps != 0 {
		t.Fatalf("black caps: %d", n.blackCaps)
	}

	n, e = n.Move(Move{2, 3, PlaceFlat, nil})
	if e != nil {
		t.Fatalf("move %v", e)
	}

	t.Log("Place too many capstones")
	_, e = n.Move(Move{0, 0, PlaceCapstone, nil})
	if e != ErrNoCapstone {
		t.Fatalf("place capstone: %v", e)
	}
	t.Log("Slide onto a capstone")
	_, e = n.Move(Move{3, 4, SlideUp, []byte{1}})
	if e != ErrIllegalSlide {
		t.Fatalf("slide onto a capstone")
	}
	t.Log("Slide a capstone onto a flat")
	n, e = n.Move(Move{3, 3, SlideDown, []byte{1}})
	if e != nil {
		t.Fatalf("cap onto flat: %v", e)
	}
	if sq := n.At(3, 4); !reflect.DeepEqual(sq,
		Square{makePiece(Black, Capstone),
			makePiece(Black, Flat)}) {
		t.Fatalf("stack wrong: %v", sq)
	}
}
