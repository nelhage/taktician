package tak

import (
	"reflect"
	"sort"
	"testing"
)

func TestMove(t *testing.T) {
	p := New(Config{Size: 5})
	p.move = 2
	p.whiteStones = 5
	p.blackStones = 5

	t.Log("Place a flat stone")
	n, e := p.Move(&Move{3, 3, PlaceFlat, nil})
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
	n, e = n.Move(&Move{3, 4, PlaceStanding, nil})
	if e != nil {
		t.Fatalf("move 2: %v", e)
	}
	if sq := n.At(3, 4); len(sq) != 1 || sq[0] != MakePiece(Black, Standing) {
		t.Fatalf("place failed: %v", sq)
	}

	t.Log("Slide onto a standing")
	orig := Move{3, 3, SlideUp, []byte{1}}
	move := orig
	_, e = n.Move(&move)
	if e != ErrIllegalSlide {
		t.Fatalf("slide onto wall allowed: %v", e)
	}
	if !reflect.DeepEqual(orig, move) {
		t.Errorf("mutated move: was=%#v now=%#v", orig, move)
	}

	t.Log("Slide onto an empty square")
	nn, e := n.Move(&Move{3, 3, SlideDown, []byte{1}})
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
	n, e = nn.Move(&Move{3, 3, PlaceCapstone, nil})
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

	n, e = n.Move(&Move{2, 3, PlaceFlat, nil})
	if e != nil {
		t.Fatalf("move %v", e)
	}

	t.Log("Place too many capstones")
	_, e = n.Move(&Move{0, 0, PlaceCapstone, nil})
	if e != ErrNoCapstone {
		t.Fatalf("place capstone: %v", e)
	}
	t.Log("Slide onto a capstone")
	_, e = n.Move(&Move{3, 4, SlideDown, []byte{1}})
	if e != ErrIllegalSlide {
		t.Fatalf("slide onto a capstone")
	}
	t.Log("Slide a capstone to flatten a wall")
	n, e = n.Move(&Move{3, 3, SlideUp, []byte{1}})
	if e != nil {
		t.Fatalf("cap onto wall: %v", e)
	}
	if sq := n.At(3, 4); !reflect.DeepEqual(sq,
		Square{MakePiece(Black, Capstone),
			MakePiece(Black, Flat)}) {
		t.Fatalf("stack wrong: %v", sq)
	}
}

func TestMoveSlideStacks(t *testing.T) {
	p := New(Config{Size: 5})
	p.move = 4
	set(p, 3, 3, Square{
		MakePiece(White, Capstone),
		MakePiece(White, Flat),
		MakePiece(Black, Flat),
	})

	next, e := p.Move(&Move{
		X: 3, Y: 3,
		Type:   SlideLeft,
		Slides: []byte{1, 1, 1}})
	if e != nil {
		t.Fatalf("slide: %v", e)
	}
	if sq := next.At(3, 3); len(sq) != 0 {
		t.Errorf("(3,3)=%v", sq)
	}
	if sq := next.At(2, 3); len(sq) != 1 || sq[0] != MakePiece(Black, Flat) {
		t.Errorf("(2,3)=%v", sq)
	}
	if sq := next.At(1, 3); len(sq) != 1 || sq[0] != MakePiece(White, Flat) {
		t.Errorf("(1,3)=%v", sq)
	}
	if sq := next.At(0, 3); len(sq) != 1 || sq[0] != MakePiece(White, Capstone) {
		t.Errorf("(0,3)=%v", sq)
	}
}

func TestMoveMultiDrop(t *testing.T) {
	p := New(Config{Size: 5})
	p.move = 4
	set(p, 1, 3, Square{
		MakePiece(White, Capstone),
		MakePiece(White, Flat),
		MakePiece(Black, Flat),
		MakePiece(Black, Flat),
		MakePiece(White, Flat),
		MakePiece(Black, Flat),
		MakePiece(Black, Flat),
	})

	next, e := p.Move(&Move{
		X: 1, Y: 3,
		Type:   SlideRight,
		Slides: []byte{2, 1, 2}})
	if e != nil {
		t.Fatalf("slide: %v", e)
	}
	expect := []struct {
		x, y int
		sq   Square
	}{
		{1, 3, Square{MakePiece(Black, Flat), MakePiece(Black, Flat)}},
		{2, 3, Square{MakePiece(Black, Flat), MakePiece(White, Flat)}},
		{3, 3, Square{MakePiece(Black, Flat)}},
		{4, 3, Square{MakePiece(White, Capstone), MakePiece(White, Flat)}},
	}
	for _, e := range expect {
		if sq := next.At(e.x, e.y); !reflect.DeepEqual(sq, e.sq) {
			t.Errorf("%d,%d=%v!=%v", e.x, e.y, sq, e.sq)
		}
	}
}

func TestAllMovesEmptyBoard(t *testing.T) {
	type coord struct{ x, y int }
	p := New(Config{Size: 6})
	moves := p.AllMoves(nil)
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
		{4, 1, []MoveType{SlideLeft, SlideDown, SlideUp}},
	}
	for _, tc := range cases {
		p := New(Config{Size: 5})
		// fake skip the opening moves
		p.move = 4
		set(p, tc.x, tc.y, Square{MakePiece(White, Flat)})
		var dirs []MoveType
		for _, m := range p.AllMoves(nil) {
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

func TestEqual(t *testing.T) {
	a := &Move{
		X: 3, Y: 4, Type: SlideDown, Slides: []byte{3},
	}
	b := &Move{
		X: 3, Y: 4, Type: SlideDown, Slides: []byte{2},
	}
	if !a.Equal(a) {
		t.Errorf("%#v != self", a)
	}
	if a.Equal(b) {
		t.Errorf("%#v = %#v!", a, b)
	}
}
