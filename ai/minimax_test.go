package ai

import (
	"flag"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

var size = flag.Int("size", 5, "board size to benchmark")
var depth = flag.Int("depth", 4, "minimax search depth")
var seed = flag.Int64("seed", 4, "random seed")

func BenchmarkMinimax(b *testing.B) {
	var cfg = tak.Config{Size: *size}
	p := tak.New(cfg)
	p, _ = p.Move(&tak.Move{X: 0, Y: 0, Type: tak.PlaceFlat})
	p, _ = p.Move(&tak.Move{X: int8(*size - 1), Y: int8(*size - 1), Type: tak.PlaceFlat})
	ai := NewMinimax(MinimaxConfig{
		Size:  *size,
		Depth: *depth,
		Seed:  *seed,
	})

	base := p.Clone()

	next := tak.Alloc(*size)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var e error
		m := ai.GetMove(context.Background(), p)
		next, e = p.MovePreallocated(&m, next)
		if e != nil {
			b.Fatal("bad move", e)
		}
		p, next = next, p

		if over, _ := p.GameOver(); over {
			p = base.Clone()
		}
	}
}

func TestRegression(t *testing.T) {
	game, err := ptn.ParseTPS(
		`2,x4/x2,2,x2/x,2,2,x2/x2,12,2,1/1,1,21,2,1 1 9`,
	)
	if err != nil {
		panic(err)
	}
	ai := NewMinimax(MinimaxConfig{Size: game.Size(), Depth: 3})
	m := ai.GetMove(context.Background(), game)
	_, e := game.Move(&m)
	if e != nil {
		t.Fatalf("ai returned illegal move: %s: %s", ptn.FormatMove(&m), e)
	}
}

func TestCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ai := NewMinimax(MinimaxConfig{Size: 5, Depth: maxDepth})
	p := tak.New(tak.Config{Size: 5})
	done := make(chan Stats)
	go func() {
		_, _, st := ai.Analyze(ctx, p)
		done <- st
	}()
	cancel()
	st := <-done
	if st.Depth == maxDepth {
		t.Fatal("wtf too deep")
	}
	if !st.Canceled {
		t.Fatal("didn't cancel")
	}
}

func TestRepeatedCancel(t *testing.T) {
	type result struct {
		ms []tak.Move
		st Stats
	}
	ctx := context.Background()
	ai := NewMinimax(MinimaxConfig{Size: 5, Depth: 6, NoNullMove: true, NoTable: true})
	p := tak.New(tak.Config{Size: 5})
	for i := 0; i < 5; i++ {
		done := make(chan result)
		start := make(chan struct{})
		ctx, cancel := context.WithCancel(ctx)
		go func() {
			start <- struct{}{}
			ms, _, st := ai.Analyze(ctx, p)
			done <- result{ms, st}
		}()
		<-start
		time.Sleep(time.Millisecond)
		cancel()
		res := <-done
		if res.st.Depth == 6 {
			t.Fatalf("[%d] cancel() didn't work", i)
		}
		if !res.st.Canceled {
			t.Fatalf("[%d] not canceled", i)
		}
		if len(res.ms) == 0 {
			t.Fatalf("[%d] canceled search did not return a move", i)
		}
	}
	ms, _, st := ai.Analyze(ctx, p)
	if len(ms) == 0 {
		t.Fatal("did not return a move")
	}
	if st.Depth != 6 {
		t.Fatal("did not do full search")
	}
}
