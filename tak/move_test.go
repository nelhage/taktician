package tak

import (
	"reflect"
	"sort"
	"testing"
)

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
	if sq := n.At(3, 3); len(sq) != 1 || sq[0] != MakePiece(White, Flat) {
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
	if sq := n.At(3, 4); len(sq) != 1 || sq[0] != MakePiece(Black, Standing) {
		t.Fatalf("place failed: %v", sq)
	}

	t.Log("Slide onto a standing")
	_, e = n.Move(Move{3, 3, SlideUp, []byte{1}})
	if e != ErrIllegalSlide {
		t.Fatalf("slide onto wall allowed: %v", e)
	}

	t.Log("Slide onto an empty square")
	nn, e := n.Move(Move{3, 3, SlideDown, []byte{1}})
	if e != nil {
		t.Fatalf("slide up: %v", e)
	}
	if sq := nn.At(3, 3); len(sq) != 0 {
		t.Fatalf("slide did not clear src: %v", sq)
	}
	if sq := nn.At(3, 2); len(sq) != 1 || sq[0] != MakePiece(White, Flat) {
		t.Fatalf("slide did not move: %v", sq)
	}
	if sq := n.At(3, 3); len(sq) != 1 || sq[0] != MakePiece(White, Flat) {
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
	if sq := n.At(3, 3); len(sq) != 1 || sq[0] != MakePiece(Black, Capstone) {
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
	_, e = n.Move(Move{3, 4, SlideDown, []byte{1}})
	if e != ErrIllegalSlide {
		t.Fatalf("slide onto a capstone")
	}
	t.Log("Slide a capstone onto a flat")
	n, e = n.Move(Move{3, 3, SlideUp, []byte{1}})
	if e != nil {
		t.Fatalf("cap onto flat: %v", e)
	}
	if sq := n.At(3, 4); !reflect.DeepEqual(sq,
		Square{MakePiece(Black, Capstone),
			MakePiece(Black, Flat)}) {
		t.Fatalf("stack wrong: %v", sq)
	}
}

func TestAllMovesEmptyBoard(t *testing.T) {
	type coord struct{ x, y int }
	p := New(Config{Size: 6})
	moves := p.AllMoves()
	lookup := make(map[coord]struct{}, 6*6)
	for _, m := range moves {
		if m.Type != PlaceFlat {
			t.Errorf("bad initial move: %#v", m)
			continue
		}
		c := coord{m.X, m.Y}
		if _, ok := lookup[c]; ok {
			t.Errorf("dup move %#v", c)
		}
		lookup[c] = struct{}{}
	}
	if len(lookup) != 6*6 {
		t.Error("wrong number of moves:", len(lookup))
	}
	for i := 0; i < 6; i++ {
		for j := 0; j < 6; j++ {
			if _, ok := lookup[coord{i, j}]; !ok {
				t.Errorf("missing move %d,%d", i, j)
			}
		}
	}
}

type orderMoves []MoveType

func (o orderMoves) Len() int {
	return len(o)
}
func (o orderMoves) Less(i, j int) bool {
	return o[i] < o[j]
}
func (o orderMoves) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}

func TestAllMovesBasicSlides(t *testing.T) {
	cases := []struct {
		x, y   int
		slides []MoveType
	}{
		{0, 0, []MoveType{SlideRight, SlideUp}},
		{0, 4, []MoveType{SlideRight, SlideDown}},
		{4, 0, []MoveType{SlideLeft, SlideUp}},
		{4, 4, []MoveType{SlideLeft, SlideDown}},
		{2, 2, []MoveType{SlideLeft, SlideDown, SlideRight, SlideUp}},
	}
	for _, tc := range cases {
		p := New(Config{Size: 5})
		// fake skip the opening moves
		p.move = 4
		p.set(tc.x, tc.y, Square{MakePiece(White, Flat)})
		var dirs []MoveType
		for _, m := range p.AllMoves() {
			if m.X != tc.x || m.Y != tc.y {
				continue
			}
			dirs = append(dirs, m.Type)
		}
		sort.Sort(orderMoves(dirs))
		sort.Sort(orderMoves(tc.slides))
		if !reflect.DeepEqual(tc.slides, dirs) {
			t.Errorf("At (%d,%d) slides=%#v want %#v",
				tc.x, tc.y, dirs, tc.slides,
			)
		}

	}
}
