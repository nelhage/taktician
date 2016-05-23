package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path"
	"reflect"
	"sync"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

var (
	size    = flag.Int("size", 5, "board size")
	zero    = flag.Bool("zero", false, "start with zero weights, not defaults")
	w1      = flag.String("w1", "", "first set of weights")
	w2      = flag.String("w2", "", "second set of weights")
	d1      = flag.Int("d1", 0, "override depth 1")
	d2      = flag.Int("d2", 0, "override depth 2")
	perturb = flag.Float64("perturb", 0.0, "perturb weights")
	seed    = flag.Int64("seed", 1, "starting seed")
	games   = flag.Int("games", 10, "number of games")
	cutoff  = flag.Int("cutoff", 81, "cut games off after how many plies")

	depth = flag.Int("depth", 3, "depth to search")
	limit = flag.Duration("limit", 0, "search duration")

	verbose = flag.Bool("verbose", false, "log results per game")

	threads = flag.Int("threads", 4, "number of parallel threads")

	out = flag.String("out", "", "directory to write ptns to")
)

type gameSpec struct {
	i            int
	white, black ai.TakPlayer
	p1color      tak.Color
}

type gameResult struct {
	spec gameSpec
	p    *tak.Position
	ms   []tak.Move
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
	if *d1 == 0 {
		*d1 = *depth
	}
	if *d2 == 0 {
		*d2 = *depth
	}

	var stats [2]struct {
		wins     int
		flatWins int
		roadWins int
	}
	var ties int

	rc := make(chan gameResult)

	go runGames(weights1, weights2, *seed, rc)
	for r := range rc {
		d := r.p.WinDetails()
		if *verbose {
			log.Printf("game n=%d plies=%d p1=%s winner=%s wf=%d bf=%d ws=%d bs=%d",
				r.spec.i, r.p.MoveNumber(),
				r.spec.p1color, d.Winner,
				d.WhiteFlats,
				d.BlackFlats,
				r.p.WhiteStones(),
				r.p.BlackStones(),
			)
		}
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
		if d.Winner == tak.NoColor {
			ties++
		}
		if *out != "" {
			writeGame(*out, &r)
		}
	}

	j, _ := json.Marshal(&weights1)
	log.Printf("p1=%s", j)
	j, _ = json.Marshal(&weights2)
	log.Printf("p2=%s", j)
	log.Printf("done games=%d seed=%d ties=%d p1.wins=%d (%d road/%d flat) p2.wins=%d (%d road/%d flat)",
		*games, *seed, ties,
		stats[0].wins, stats[0].roadWins, stats[0].flatWins,
		stats[1].wins, stats[1].roadWins, stats[1].flatWins)
	a, b := int64(stats[0].wins), int64(stats[1].wins)
	if a < b {
		a, b = b, a
	}
	log.Printf("p[one-sided]=%f", binomTest(a, b, 0.5))
}

func writeGame(d string, r *gameResult) {
	os.MkdirAll(d, 0755)
	p := &ptn.PTN{}
	p.Tags = []ptn.Tag{
		{"Size", fmt.Sprintf("%d", r.p.Size())},
		{"Player1", r.spec.p1color.String()},
	}
	for i, m := range r.ms {
		if i%2 == 0 {
			p.Ops = append(p.Ops, &ptn.MoveNumber{Number: i/2 + 1})
		}
		p.Ops = append(p.Ops, &ptn.Move{Move: m})
	}
	ptnPath := path.Join(d, fmt.Sprintf("%d.ptn", r.spec.i))
	ioutil.WriteFile(ptnPath, []byte(p.Render()), 0644)
}

func worker(games <-chan gameSpec, out chan<- gameResult) {
	for g := range games {
		var ms []tak.Move
		p := tak.New(tak.Config{Size: *size})
		for i := 0; i < *cutoff; i++ {
			var m tak.Move
			if p.ToMove() == tak.White {
				m = g.white.GetMove(p, *limit)
			} else {
				m = g.black.GetMove(p, *limit)
			}
			p, _ = p.Move(&m)
			ms = append(ms, m)
			if ok, _ := p.GameOver(); ok {
				break
			}
		}
		out <- gameResult{
			spec: g,
			p:    p,
			ms:   ms,
		}
	}
}

func perturbWeights(p float64, w ai.Weights) ai.Weights {
	r := reflect.Indirect(reflect.ValueOf(&w))
	typ := r.Type()
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.Type.Kind() != reflect.Int {
			continue
		}
		v := r.Field(i).Interface().(int)
		adj := rand.NormFloat64() * p
		v = int(float64(v) * (1 + adj))
		r.Field(i).SetInt(int64(v))
	}

	j, _ := json.Marshal(&w)
	log.Printf("perturb: %s", j)

	return w
}

func runGames(w1, w2 ai.Weights, seed int64, rc chan<- gameResult) {
	gc := make(chan gameSpec)
	var wg sync.WaitGroup
	wg.Add(*threads)
	for i := 0; i < *threads; i++ {
		go func() {
			worker(gc, rc)
			wg.Done()
		}()
	}
	r := rand.New(rand.NewSource(seed))
	for g := 0; g < *games; g++ {
		var white, black ai.TakPlayer
		w1 := w1
		w2 := w2
		if *perturb != 0.0 {
			w1 = perturbWeights(*perturb, w1)
			w2 = perturbWeights(*perturb, w2)
		}
		p1 := ai.NewMinimax(ai.MinimaxConfig{
			Depth:    *d1,
			Seed:     r.Int63(),
			Evaluate: ai.MakeEvaluator(&w1),
			Size:     *size,
		})
		p2 := ai.NewMinimax(ai.MinimaxConfig{
			Depth:    *d2,
			Seed:     r.Int63(),
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
