package ai

import (
	"math/rand"
	"testing"

	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
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

	p := tak.New(tak.Config{Size: 6})
	r := rand.New(rand.NewSource(1))
	m, ok := ob.GetMove(p, r)
	if !ok {
		t.Fatal("no move")
	}
	f := ptn.FormatMove(m)
	if f != "a1" {
		t.Fatal("wrong move: ", f)
	}

}
