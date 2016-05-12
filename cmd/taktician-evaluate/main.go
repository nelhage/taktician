package main

import (
	"encoding/json"
	"flag"
	"log"
	"sync"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/tak"
)

var (
	depth  = flag.Int("depth", 3, "depth to search")
	size   = flag.Int("size", 5, "board size")
	zero   = flag.Bool("zero", false, "start with zero weights, not defaults")
	w1     = flag.String("w1", "", "first set of weights")
	w2     = flag.String("w2", "", "second set of weights")
	seed   = flag.Int64("seed", 1, "starting seed")
	games  = flag.Int("games", 10, "number of games")
	cutoff = flag.Int("cutoff", 81, "cut games off after how many plies")
)

type gameSpec struct {
	i            int
	white, black ai.TakPlayer
	p1color      tak.Color
}

type gameResult struct {
	spec gameSpec
	p    *tak.Position
}

func main() {
	flag.Parse()

	weights1 := ai.DefaultWeights
	weights2 := ai.DefaultWeights
	if *zero {
		weights1 = ai.Weights{}
		weights2 = ai.Weights{}
	}
	if *w1 != "" {
		err := json.Unmarshal([]byte(*w1), &weights1)
		if err != nil {
			log.Fatal("w1:", err)
		}
	}
	if *w2 != "" {
		err := json.Unmarshal([]byte(*w2), &weights2)
		if err != nil {
			log.Fatal("w2:", err)
		}
	}

	var stats [2]struct {
		wins     int
		flatWins int
		roadWins int
	}

	rc := make(chan gameResult)

	go runGames(weights1, weights2, *seed, rc)
	for r := range rc {
		d := r.p.WinDetails()
		log.Printf("game n=%d plies=%d p1=%s winner=%s wf=%d bf=%d ws=%d bs=%d",
			r.spec.i, r.p.MoveNumber(),
			r.spec.p1color, d.Winner,
			d.WhiteFlats,
			d.BlackFlats,
			r.p.WhiteStones(),
			r.p.BlackStones(),
		)
		if d.Over {
			st := &stats[0]
			if d.Winner == r.spec.p1color.Flip() {
				st = &stats[1]
			}
			st.wins++
			switch d.Reason {
			case tak.FlatsWin:
				st.flatWins++
			case tak.RoadWin:
				st.roadWins++
			}
		}
	}

	log.Printf("done games=%d seed=%d p1.wins=%d (%d road/%d flat) p2.wins=%d (%d road/%d flat)",
		*games, *seed,
		stats[0].wins, stats[0].roadWins, stats[0].flatWins,
		stats[1].wins, stats[1].roadWins, stats[1].flatWins)
}

func worker(games <-chan gameSpec, out chan<- gameResult) {
	for g := range games {
		p := tak.New(tak.Config{Size: *size})
		for i := 0; i < *cutoff; i++ {
			var m tak.Move
			if p.ToMove() == tak.White {
				m = g.white.GetMove(p, 0)
			} else {
				m = g.black.GetMove(p, 0)
			}
			p, _ = p.Move(&m)
			if ok, _ := p.GameOver(); ok {
				break
			}
		}
		out <- gameResult{
			spec: g,
			p:    p,
		}
	}
}

func runGames(w1, w2 ai.Weights, seed int64, rc chan<- gameResult) {
	gc := make(chan gameSpec)
	var wg sync.WaitGroup
	wg.Add(4)
	for i := 0; i < 4; i++ {
		go func() {
			worker(gc, rc)
			wg.Done()
		}()
	}
	for g := 0; g < *games; g++ {
		var white, black ai.TakPlayer
		p1 := ai.NewMinimax(ai.MinimaxConfig{
			Depth:    *depth,
			Seed:     seed,
			Evaluate: ai.MakeEvaluator(&w1),
			Size:     *size,
		})
		seed++
		p2 := ai.NewMinimax(ai.MinimaxConfig{
			Depth:    *depth,
			Seed:     seed,
			Evaluate: ai.MakeEvaluator(&w2),
			Size:     *size,
		})
		seed++
		var p1color tak.Color
		if g%2 == 0 {
			white, black = p1, p2
			p1color = tak.White
		} else {
			black, white = p1, p2
			p1color = tak.Black
		}

		spec := gameSpec{
			i:       g,
			white:   white,
			black:   black,
			p1color: p1color,
		}
		gc <- spec
	}
	close(gc)
	wg.Wait()
	close(rc)
}
