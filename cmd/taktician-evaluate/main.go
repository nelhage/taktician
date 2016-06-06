package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime/pprof"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

var (
	size    = flag.Int("size", 5, "board size")
	zero    = flag.Bool("zero", false, "start with zero weights, not defaults")
	w1      = flag.String("w1", "", "first set of weights")
	w2      = flag.String("w2", "", "second set of weights")
	c1      = flag.String("c1", "", "custom config 1")
	c2      = flag.String("c2", "", "custom config 2")
	perturb = flag.Float64("perturb", 0.0, "perturb weights")
	seed    = flag.Int64("seed", 1, "starting random seed")
	games   = flag.Int("games", 10, "number of games to play")
	cutoff  = flag.Int("cutoff", 81, "cut games off after how many plies")
	swap    = flag.Bool("swap", true, "swap colors each game")

	prefix = flag.String("prefix", "", "ptn file to start games at the end of")

	depth = flag.Int("depth", 3, "depth to search each move")
	limit = flag.Duration("limit", 0, "amount of time to search each move")

	threads = flag.Int("threads", 4, "number of parallel threads")

	out = flag.String("out", "", "directory to write ptns to")

	search = flag.Bool("search", false, "search for a good set of weights")

	memProfile = flag.String("mem-profile", "", "write memory profile")
)

func main() {
	flag.Parse()
	if *memProfile != "" {
		defer func() {
			f, e := os.OpenFile(*memProfile,
				os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
			if e != nil {
				log.Printf("open memory profile: %v", e)
				return
			}
			pprof.Lookup("heap").WriteTo(f, 0)
		}()
	}

	var p *tak.Position
	if *prefix != "" {
		bs, e := ioutil.ReadFile(*prefix)
		if e != nil {
			log.Fatalf("Read %s: %v", *prefix, e)
		}
		pt, e := ptn.ParsePTN(bytes.NewBuffer(bs))
		if e != nil {
			log.Fatalf("Parse PTN: %v", e)
		}
		p, e = pt.PositionAtMove(0, tak.NoColor)
		if e != nil {
			log.Fatalf("PTN: %v", e)
		}
	}

	weights1 := ai.DefaultWeights[*size]
	weights2 := ai.DefaultWeights[*size]
	if *zero {
		weights1 = ai.Weights{}
		weights2 = ai.Weights{}
	}
	if *w1 != "" {
		if err := json.Unmarshal([]byte(*w1), &weights1); err != nil {
			log.Fatal("w1:", err)
		}
	}
	if *w2 != "" {
		if err := json.Unmarshal([]byte(*w2), &weights2); err != nil {
			log.Fatal("w2:", err)
		}
	}

	cfg1 := ai.MinimaxConfig{
		Depth: *depth,
		Size:  *size,
	}
	cfg2 := ai.MinimaxConfig{
		Depth: *depth,
		Size:  *size,
	}
	if *c1 != "" {
		if err := json.Unmarshal([]byte(*c1), &cfg1); err != nil {
			log.Fatal("c1:", err)
		}
	}
	if *c2 != "" {
		if err := json.Unmarshal([]byte(*c2), &cfg2); err != nil {
			log.Fatal("c2:", err)
		}
	}

	if *search {
		doSearch(cfg1, weights1)
		return
	}

	st := Simulate(&Config{
		Cfg1:    cfg1,
		Cfg2:    cfg2,
		W1:      weights1,
		W2:      weights2,
		Swap:    *swap,
		Games:   *games,
		Threads: *threads,
		Seed:    *seed,
		Cutoff:  *cutoff,
		Limit:   *limit,
		Perturb: *perturb,
		Initial: p,
	})

	if *out != "" {
		for _, r := range st.Games {
			writeGame(*out, &r)
		}
	}

	var j []byte
	j, _ = json.Marshal(&weights1)
	log.Printf("p1w=%s", j)
	if *c1 != "" {
		log.Printf("p1c=!%s", *c1)
	}
	j, _ = json.Marshal(&weights2)
	log.Printf("p2w=%s", j)
	if *c2 != "" {
		log.Printf("p2c=!%s", *c2)
	}
	log.Printf("done games=%d seed=%d ties=%d p1.wins=%d (%d road/%d flat) p2.wins=%d (%d road/%d flat)",
		*games, *seed, st.Ties,
		st.Players[0].Wins, st.Players[0].RoadWins, st.Players[0].FlatWins,
		st.Players[1].Wins, st.Players[1].RoadWins, st.Players[1].FlatWins)
	a, b := int64(st.Players[0].Wins), int64(st.Players[1].Wins)
	if a < b {
		a, b = b, a
	}
	log.Printf("white=%d black=%d (%.2f)",
		st.White, st.Black, float64(st.White)/float64(st.White+st.Black))
	log.Printf("p[one-sided]=%f", binomTest(a, b, 0.5))
}

func writeGame(d string, r *Result) {
	os.MkdirAll(d, 0755)
	p := &ptn.PTN{}
	p.Tags = []ptn.Tag{
		{"Size", fmt.Sprintf("%d", r.Position.Size())},
		{"Player1", r.spec.p1color.String()},
	}
	for i, m := range r.Moves {
		if i%2 == 0 {
			p.Ops = append(p.Ops, &ptn.MoveNumber{Number: i/2 + 1})
		}
		p.Ops = append(p.Ops, &ptn.Move{Move: m})
	}
	ptnPath := path.Join(d, fmt.Sprintf("%d.ptn", r.spec.i))
	ioutil.WriteFile(ptnPath, []byte(p.Render()), 0644)
}
