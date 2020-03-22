package tak

import "testing"

func TestPositionEqual(t *testing.T) {
	p := New(Config{Size: 5})
	if !p.Equal(p) {
		t.Fatal("New() != self!")
	}
	p2 := New(Config{Size: 5})
	if !p.Equal(p2) {
		t.Fatal("New() != New()!")
	}
	l := moves([]Move{
		Move{X: 0, Y: 0, Type: PlaceFlat},
		Move{X: 4, Y: 4, Type: PlaceFlat},
		Move{X: 0, Y: 4, Type: PlaceFlat},
		Move{X: 4, Y: 0, Type: PlaceFlat},
	})
	r := moves([]Move{
		Move{X: 4, Y: 0, Type: PlaceFlat},
		Move{X: 0, Y: 4, Type: PlaceFlat},
		Move{X: 4, Y: 4, Type: PlaceFlat},
		Move{X: 0, Y: 0, Type: PlaceFlat},
	})
	if !l.Equal(r) {
		t.Fatalf("l != r")
	}
	if !r.Equal(l) {
		t.Fatalf("r != l")
	}
	if p.Equal(r) {
		t.Fatalf("New() == r")
	}
	if p.Equal(l) {
		t.Fatalf("New() == l")
	}

}
