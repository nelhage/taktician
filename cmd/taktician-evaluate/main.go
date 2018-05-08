package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

var (
	size   = flag.Int("size", 5, "board size")
	zero   = flag.Bool("zero", false, "start with zero weights, not defaults")
	p1     = flag.String("p1", "minimax", "player1 AI engine")
	p2     = flag.String("p2", "minimax", "player2 AI engine")
	w1     = flag.String("w1", "", "first set of weights")
	w2     = flag.String("w2", "", "second set of weights")
	c1     = flag.String("c1", "", "custom config 1")
	c2     = flag.String("c2", "", "custom config 2")
	seed   = flag.Int64("seed", 0, "starting random seed")
	games  = flag.Int("games", 10, "number of games to play per opening/color")
	cutoff = flag.Int("cutoff", 80, "cut games off after how many plies")
	swap   = flag.Bool("swap", true, "swap colors each game")

	prefix = flag.String("prefix", "", "ptn file to start games at the end of")
	seeds  = flag.String("seeds", "", "directory of seed positions")

	debug = flag.Int("debug", 0, "debug level")
	depth = flag.Int("depth", 3, "depth to search each move")
	limit = flag.Duration("limit", 0, "amount of time to search each move")

	threads = flag.Int("threads", 4, "number of parallel threads")

	out     = flag.String("out", "", "directory to write ptns to")
	verbose = flag.Bool("v", false, "verbose output")

	memProfile = flag.String("mem-profile", "", "write memory profile")
)

func addSeeds(g *ptn.PTN, ps []*tak.Position) ([]*tak.Position, error) {
	p, e := g.PositionAtMove(0, tak.NoColor)
	if e != nil {
		return nil, e
	}
	return append(ps, p), nil
}

func readSeeds(d string) ([]*tak.Position, error) {
	ents, e := ioutil.ReadDir(d)
	if e != nil {
		return nil, e
	}
	var ps []*tak.Position
	for _, de := range ents {
		if !strings.HasSuffix(de.Name(), ".ptn") {
			continue
		}
		g, e := ptn.ParseFile(path.Join(d, de.Name()))
		if e != nil {
			return nil, fmt.Errorf("%s/%s: %v", d, de.Name(), e)
		}
		ps, e = addSeeds(g, ps)
		if e != nil {
			return nil, fmt.Errorf("%s/%s: %v", d, de.Name(), e)
		}
	}
	return ps, nil
}

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

	if *seed == 0 {
		*seed = time.Now().Unix()
	}

	var starts []*tak.Position
	if *prefix != "" {
		pt, e := ptn.ParseFile(*prefix)
		if e != nil {
			log.Fatalf("Parse PTN: %v", e)
		}
		p, e := pt.PositionAtMove(0, tak.NoColor)
		if e != nil {
			log.Fatalf("PTN: %v", e)
		}
		starts = []*tak.Position{p}
	}
	if *seeds != "" {
		var e error
		starts, e = readSeeds(*seeds)
		if e != nil {
			log.Fatalf("-seeds: %v", e)
		}
	}
	if len(starts) == 0 {
		starts = []*tak.Position{tak.New(tak.Config{Size: *size})}
	}

	cfg := &Config{
		Zero:    *zero,
		Size:    *size,
		Depth:   *depth,
		Debug:   *debug,
		Swap:    *swap,
		Games:   *games,
		Threads: *threads,
		Seed:    *seed,
		Cutoff:  *cutoff,
		Limit:   *limit,
		Initial: starts,
		Verbose: *verbose,
	}
	cfg.F1 = buildFactory(cfg, *p1, *c1, *w1)
	cfg.F2 = buildFactory(cfg, *p2, *c2, *w2)

	st := Simulate(cfg)

	if *out != "" {
		for _, r := range st.Games {
			writeGame(*out, &r)
		}
	}

	log.Printf("done games=%d seed=%d ties=%d cutoff=%d white=%d black=%d depth=%d limit=%s",
		len(st.Games), *seed, st.Ties, st.Cutoff, st.White, st.Black, *depth, *limit)
	log.Printf("p1.wins=%d (%d road/%d flat) p2.wins=%d (%d road/%d flat)",
		st.Players[0].Wins, st.Players[0].RoadWins, st.Players[0].FlatWins,
		st.Players[1].Wins, st.Players[1].RoadWins, st.Players[1].FlatWins)
	a, b := int64(st.Players[0].Wins), int64(st.Players[1].Wins)
	if a < b {
		a, b = b, a
	}
	log.Printf("p[one-sided]=%f", binomTest(a, b, 0.5))
}

func writeGame(d string, r *Result) {
	os.MkdirAll(d, 0755)
	p := &ptn.PTN{}
	p.Tags = []ptn.Tag{
		{Name: "Size", Value: fmt.Sprintf("%d", r.Position.Size())},
		{Name: "Player1", Value: r.spec.p1color.String()},
	}
	if r.Initial.MoveNumber() != 0 {
		p.Tags = append(p.Tags, ptn.Tag{
			Name: "TPS", Value: ptn.FormatTPS(r.Initial)})
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
