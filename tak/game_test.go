package tak

import "testing"

func TestHasRoad(t *testing.T) {
	p := New(Config{Size: 5})

	p.analyze()
	_, ok := p.hasRoad()
	if ok {
		t.Errorf("empty board hasRoad!")
	}

	for y := 0; y < 5; y++ {
		set(p, 2, y, Square{MakePiece(Black, Flat)})
	}

	p.analyze()
	c, ok := p.hasRoad()
	if !ok || c != Black {
		t.Errorf("c=%v hasRoad=%v\n", c, ok)
	}

	set(p, 2, 0, nil)
	set(p, 1, 0, Square{MakePiece(Black, Flat)})
	set(p, 1, 1, Square{MakePiece(Black, Flat)})

	p.analyze()
	c, ok = p.hasRoad()
	if !ok || c != Black {
		t.Errorf("c=%v hasRoad=%v\n", c, ok)
	}

	set(p, 1, 1, Square{MakePiece(Black, Standing)})
	p.analyze()
	c, ok = p.hasRoad()
	if ok {
		t.Errorf("c=%v hasRoad=%v\n", c, ok)
	}

	p = New(Config{Size: 5})
	set(p, 0, 1, Square{MakePiece(White, Flat)})
	set(p, 1, 1, Square{MakePiece(White, Flat)})
	set(p, 1, 2, Square{MakePiece(White, Flat)})
	set(p, 2, 2, Square{MakePiece(White, Flat)})
	set(p, 2, 3, Square{MakePiece(White, Flat)})
	set(p, 3, 3, Square{MakePiece(White, Flat)})
	set(p, 3, 4, Square{MakePiece(White, Flat)})
	set(p, 4, 4, Square{MakePiece(White, Flat)})

	p.analyze()
	c, ok = p.hasRoad()
	if !ok || c != White {
		t.Errorf("c=%v hasRoad=%v\n", c, ok)
	}
}

func TestHasRoadRegression(t *testing.T) {
	p := New(Config{Size: 5})
	set(p, 1, 4, Square{MakePiece(White, Flat)})
	set(p, 1, 3, Square{MakePiece(White, Flat)})
	set(p, 1, 2, Square{MakePiece(White, Flat)})
	set(p, 2, 2, Square{MakePiece(White, Flat)})
	set(p, 3, 2, Square{MakePiece(White, Flat)})
	set(p, 4, 2, Square{MakePiece(White, Flat)})
	set(p, 4, 1, Square{MakePiece(White, Flat)})
	set(p, 4, 0, Square{MakePiece(White, Flat)})
	p.analyze()
	c, ok := p.hasRoad()
	if !ok || c != White {
		t.Errorf("c=%v hasRoad=%v\n", c, ok)
	}
}

func TestFlatsWinner(t *testing.T) {
	p := New(Config{Size: 5})
	set(p, 0, 0, Square{MakePiece(White, Flat)})
	w := p.flatsWinner()
	if w != White {
		t.Fatal("flats winner", p)
	}
	set(p, 1, 1, Square{MakePiece(Black, Flat),
		MakePiece(White, Flat)})

	set(p, 1, 2, Square{MakePiece(Black, Flat)})
	w = p.flatsWinner()
	if w != Black {
		t.Fatal("flats winner", p)
	}
	set(p, 1, 3, Square{MakePiece(White, Flat)})
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
	b.ReportAllocs()
	b.ResetTimer()
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
			set(p, i, j, Square{piece})
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.analyze()
		p.hasRoad()
	}
}

func BenchmarkHasRoadWindy(b *testing.B) {
	p := New(Config{Size: 5})
	for y := 0; y < 4; y++ {
		set(p, 3, y, Square{MakePiece(White, Flat)})
	}
	for x := 0; x < 4; x++ {
		set(p, x, 3, Square{MakePiece(White, Flat)})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.analyze()
		p.hasRoad()
	}
}

func moves(ms []Move) *Position {
	p := New(Config{Size: 5})
	for _, m := range ms {
		n, e := p.Move(&m)
		if e != nil {
			panic("move")
		}
		p = n
	}
	return p
}

func TestHash(t *testing.T) {
	// t.SkipNow()
	a := moves([]Move{
		Move{X: 0, Y: 0, Type: PlaceFlat},
		Move{X: 1, Y: 1, Type: PlaceFlat},

		Move{X: 2, Y: 2, Type: PlaceFlat},
		Move{X: 3, Y: 3, Type: PlaceFlat},

		Move{X: 1, Y: 3, Type: PlaceFlat},
		Move{X: 3, Y: 1, Type: PlaceFlat},
	})

	b := moves([]Move{
		Move{X: 0, Y: 0, Type: PlaceFlat},
		Move{X: 1, Y: 1, Type: PlaceFlat},

		Move{X: 1, Y: 3, Type: PlaceFlat},
		Move{X: 3, Y: 1, Type: PlaceFlat},

		Move{X: 2, Y: 2, Type: PlaceFlat},
		Move{X: 3, Y: 3, Type: PlaceFlat},
	})

	if a.Hash() != b.Hash() {
		t.Fatalf("hashes don't match")
	}

	c := moves([]Move{
		Move{X: 0, Y: 0, Type: PlaceFlat},
		Move{X: 1, Y: 1, Type: PlaceFlat},

		Move{X: 1, Y: 3, Type: PlaceFlat},
		Move{X: 3, Y: 1, Type: PlaceFlat},
	})
	if c.Hash() == a.Hash() {
		t.Fatalf("collision ah=%x ch=%x aH=%x cH=%x\na=%x %x %x %x\nb=%x %x %x %x",
			a.hash, c.hash, a.Hash(), c.Hash(),
			a.White, a.Black, a.Standing, a.Caps,
			c.White, c.Black, c.Standing, c.Caps,
		)
	}

	d := moves([]Move{
		Move{X: 0, Y: 0, Type: PlaceFlat},
		Move{X: 1, Y: 1, Type: PlaceFlat},

		Move{X: 3, Y: 2, Type: PlaceFlat},
		Move{X: 4, Y: 3, Type: PlaceFlat},

		Move{X: 1, Y: 3, Type: PlaceFlat},
		Move{X: 3, Y: 1, Type: PlaceFlat},

		Move{X: 3, Y: 2, Type: SlideLeft, Slides: []byte{1}},
		Move{X: 4, Y: 3, Type: SlideLeft, Slides: []byte{1}},
	})
	if d.Hash() != a.Hash() {
		t.Fatalf("hash fail")
	}

	e := moves([]Move{
		Move{X: 0, Y: 0, Type: PlaceFlat},
		Move{X: 1, Y: 1, Type: PlaceFlat},

		Move{X: 1, Y: 2, Type: PlaceFlat},
		Move{X: 2, Y: 1, Type: PlaceStanding},
	})
	f := moves([]Move{
		Move{X: 0, Y: 0, Type: PlaceFlat},
		Move{X: 1, Y: 1, Type: PlaceFlat},

		Move{X: 1, Y: 2, Type: PlaceStanding},
		Move{X: 2, Y: 1, Type: PlaceFlat},
	})

	if e.Hash() == f.Hash() {
		t.Fatalf("hash fail when swapping flat/standing")
	}
}
