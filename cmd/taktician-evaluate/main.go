package main

import (
	"encoding/json"
	"flag"
	"log"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/tak"
)

var (
	depth  = flag.Int("depth", 3, "depth to search")
	size   = flag.Int("size", 5, "board size")
	w1     = flag.String("w1", "", "first set of weights")
	w2     = flag.String("w2", "", "second set of weights")
	seed   = flag.Int("seed", 1, "starting seed")
	games  = flag.Int("games", 10, "number of games")
	cutoff = flag.Int("cutoff", 81, "cut games off after how many plies")
)

func main() {
	flag.Parse()

	weights1 := ai.DefaultWeights
	weights2 := ai.DefaultWeights
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

	for g := 0; g < *games; g++ {
		var white, black ai.TakPlayer
		p1 := ai.NewMinimax(ai.MinimaxConfig{
			Depth:    *depth,
			Seed:     int64(*seed),
			Evaluate: ai.MakeEvaluator(&weights1),
			Size:     *size,
		})
		*seed++
		p2 := ai.NewMinimax(ai.MinimaxConfig{
			Depth:    *depth,
			Seed:     int64(*seed),
			Evaluate: ai.MakeEvaluator(&weights2),
			Size:     *size,
		})
		*seed++
		var p1color tak.Color
		if g%2 == 0 {
			white, black = p1, p2
			p1color = tak.White
		} else {
			black, white = p1, p2
			p1color = tak.Black
		}

		p := tak.New(tak.Config{Size: *size})
		for i := 0; i < *cutoff; i++ {
			var m tak.Move
			if p.ToMove() == tak.White {
				m = white.GetMove(p, 0)
			} else {
				m = black.GetMove(p, 0)
			}
			p, _ = p.Move(&m)
			if ok, _ := p.GameOver(); ok {
				break
			}
		}
		d := p.WinDetails()
		log.Printf("game n=%d plies=%d p1=%s winner=%s wf=%d bf=%d ws=%d bs=%d",
			g, p.MoveNumber(),
			p1color, d.Winner,
			d.WhiteFlats,
			d.BlackFlats,
			p.WhiteStones(),
			p.BlackStones(),
		)
		if d.Over {
			st := &stats[0]
			if d.Winner == p1color.Flip() {
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
	log.Printf("done games=%d p1.wins=%d (%d road/%d flat) p2.wins=%d (%d road/%d flat)",
		*games,
		stats[0].wins, stats[0].roadWins, stats[0].flatWins,
		stats[1].wins, stats[1].roadWins, stats[1].flatWins)
}
