package tak

import "testing"

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
		p.board[y*5+2] = Square{MakePiece(Black, Flat)}
	}
	c, ok := p.hasRoad()
	if !ok || c != Black {
		t.Errorf("c=%v hasRoad=%v\n", c, ok)
	}

	p.set(2, 0, nil)
	p.set(1, 0, Square{MakePiece(Black, Flat)})
	p.set(1, 1, Square{MakePiece(Black, Flat)})
	c, ok = p.hasRoad()
	if !ok || c != Black {
		t.Errorf("c=%v hasRoad=%v\n", c, ok)
	}

	p.set(1, 1, Square{MakePiece(Black, Standing)})
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
	ok, _ := p.GameOver()
	if ok {
		t.Fatalf("over, but capstone is left")
	}
}

func BenchmarkEmptyHasRoad(b *testing.B) {
	p := New(Config{Size: 5})
	for i := 0; i < b.N; i++ {
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
		p.hasRoad()
	}
}
