package game

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
