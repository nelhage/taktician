package ai

import (
	"math/rand"
	"testing"

	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/taktest"
)

func TestOpeningBook(t *testing.T) {
	moves := []string{
		`a1 f1`,
		`a1 f6`,
	}
	ob, err := BuildOpeningBook(6, moves)
	if err != nil {
		t.Fatal("build: ", err)
	}

	r := rand.New(rand.NewSource(1))

	p := taktest.Position(6, "")
	m, ok := ob.GetMove(p, r)
	if !ok {
		t.Fatal("no move")
	}
	f := ptn.FormatMove(m)
	if f != "a1" {
		t.Fatal("wrong move: ", f)
	}

	p = taktest.Position(6, "f1")
	m, ok = ob.GetMove(p, r)
	if !ok {
		t.Fatal("no move f1")
	}

	pos := ob.book[p.Hash()]
	if len(pos.moves) != 2 {
		t.Fatal("wrong children n=", len(pos.moves))
	}
}

func TestCollisions(t *testing.T) {
	ob, err := BuildOpeningBook(6, []string{`a1 f6 d4 d3 c4`})
	if err != nil {
		t.Fatal("build ", err)
	}
	p := taktest.Position(6, "a1 f6 d4 d3")
	pos := ob.book[p.Hash()]
	if pos == nil {
		t.Fatal("did not store")
	}
	for _, c := range pos.moves {
		_, e := p.Move(c.move)
		if e != nil {
			t.Logf("children=%#v", pos.moves)
			t.Errorf("illegal move=%s w=%d", ptn.FormatMove(c.move), c.weight)
		}
	}
}
