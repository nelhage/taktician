package tak

import (
	"strconv"
	"testing"
)

func TestPrecompute(t *testing.T) {
	c := &Config{Size: 5}
	c.precompute()
	if c.b != (1<<5)-1 {
		t.Error("c.b(5):", strconv.FormatUint(c.b, 2))
	}
	if c.t != ((1<<5)-1)<<(4*5) {
		t.Error("c.t(5):", strconv.FormatUint(c.t, 2))
	}
	if c.r != 0x0108421 {
		t.Error("c.r(5):", strconv.FormatUint(c.r, 2))
	}
	if c.l != 0x1084210 {
		t.Error("c.l(5):", strconv.FormatUint(c.l, 2))
	}
	if c.mask != 0x1ffffff {
		t.Error("c.mask(5):", strconv.FormatUint(c.mask, 2))
	}

	c = &Config{Size: 8}
	c.precompute()
	if c.b != (1<<8)-1 {
		t.Error("c.b(8):", strconv.FormatUint(c.b, 2))
	}
	if c.t != ((1<<8)-1)<<(7*8) {
		t.Error("c.t(8):", strconv.FormatUint(c.t, 2))
	}
	if c.r != 0x101010101010101 {
		t.Error("c.r(8):", strconv.FormatUint(c.r, 2))
	}
	if c.l != 0x8080808080808080 {
		t.Error("c.l(8):", strconv.FormatUint(c.l, 2))
	}
	if c.mask != ^uint64(0) {
		t.Error("c.mask(8):", strconv.FormatUint(c.mask, 2))
	}
}

func TestHasRoad(t *testing.T) {
	p := New(Config{Size: 5})

	p.analyze()
	_, ok := p.hasRoad()
	if ok {
		t.Errorf("empty board hasRoad!")
	}

	for y := 0; y < 5; y++ {
		p.board[y*5+2] = Square{MakePiece(Black, Flat)}
	}

	p.analyze()
	c, ok := p.hasRoad()
	if !ok || c != Black {
		t.Errorf("c=%v hasRoad=%v\n", c, ok)
	}

	p.set(2, 0, nil)
	p.set(1, 0, Square{MakePiece(Black, Flat)})
	p.set(1, 1, Square{MakePiece(Black, Flat)})

	p.analyze()
	c, ok = p.hasRoad()
	if !ok || c != Black {
		t.Errorf("c=%v hasRoad=%v\n", c, ok)
	}

	p.set(1, 1, Square{MakePiece(Black, Standing)})
	p.analyze()
	c, ok = p.hasRoad()
	if ok {
		t.Errorf("c=%v hasRoad=%v\n", c, ok)
	}

	p.board = make([]Square, 5*5)
	p.set(0, 1, Square{MakePiece(White, Flat)})
	p.set(1, 1, Square{MakePiece(White, Flat)})
	p.set(1, 2, Square{MakePiece(White, Flat)})
	p.set(2, 2, Square{MakePiece(White, Flat)})
	p.set(2, 3, Square{MakePiece(White, Flat)})
	p.set(3, 3, Square{MakePiece(White, Flat)})
	p.set(3, 4, Square{MakePiece(White, Flat)})
	p.set(4, 4, Square{MakePiece(White, Flat)})

	p.analyze()
	c, ok = p.hasRoad()
	if !ok || c != White {
		t.Errorf("c=%v hasRoad=%v\n", c, ok)
	}
}

func TestHasRoadRegression(t *testing.T) {
	p := New(Config{Size: 5})
	p.set(1, 4, Square{MakePiece(White, Flat)})
	p.set(1, 3, Square{MakePiece(White, Flat)})
	p.set(1, 2, Square{MakePiece(White, Flat)})
	p.set(2, 2, Square{MakePiece(White, Flat)})
	p.set(3, 2, Square{MakePiece(White, Flat)})
	p.set(4, 2, Square{MakePiece(White, Flat)})
	p.set(4, 1, Square{MakePiece(White, Flat)})
	p.set(4, 0, Square{MakePiece(White, Flat)})
	p.analyze()
	c, ok := p.hasRoad()
	if !ok || c != White {
		t.Errorf("c=%v hasRoad=%v\n", c, ok)
	}
}

func TestFlatsWinner(t *testing.T) {
	p := New(Config{Size: 5})
	p.set(0, 0, Square{MakePiece(White, Flat)})
	w := p.flatsWinner()
	if w != White {
		t.Fatal("flats winner", p)
	}
	p.set(1, 1, Square{MakePiece(Black, Flat),
		MakePiece(White, Flat)})
	p.set(1, 2, Square{MakePiece(Black, Flat)})
	w = p.flatsWinner()
	if w != Black {
		t.Fatal("flats winner", p)
	}
	p.set(1, 3, Square{MakePiece(White, Flat)})
	w = p.flatsWinner()
	if w != NoColor {
		t.Fatal("flats winner", p)
	}
}

func TestFlatsWinnerCapLeft(t *testing.T) {
	p := New(Config{Size: 5})
	p.whiteStones = 0
	p.analyze()
	ok, _ := p.GameOver()
	if ok {
		t.Fatalf("over, but capstone is left")
	}
}

func BenchmarkEmptyHasRoad(b *testing.B) {
	p := New(Config{Size: 5})
	for i := 0; i < b.N; i++ {
		p.analyze()
		p.hasRoad()
	}
}

func BenchmarkFullHasRoad(b *testing.B) {
	p := New(Config{Size: 5})
	for i := 0; i < p.Size(); i++ {
		for j := 0; j < p.Size(); j++ {
			var piece Piece
			if (i^j)&1 == 0 {
				piece = MakePiece(White, Flat)
			} else {
				piece = MakePiece(Black, Flat)
			}
			p.set(i, j, Square{piece})
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.analyze()
		p.hasRoad()
	}
}

func BenchmarkHasRoadWindy(b *testing.B) {
	p := New(Config{Size: 5})
	for y := 0; y < 4; y++ {
		p.set(3, y, Square{MakePiece(White, Flat)})
	}
	for x := 0; x < 4; x++ {
		p.set(x, 3, Square{MakePiece(White, Flat)})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.analyze()
		p.hasRoad()
	}
}
