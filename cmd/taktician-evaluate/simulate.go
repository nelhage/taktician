package main

import (
	"fmt"
	"log"
	"math/rand"
	"reflect"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

type Config struct {
	Games int

	Verbose bool

	Initial []*tak.Position

	Cfg1, Cfg2 ai.MinimaxConfig
	W1, W2     ai.Weights

	Swap    bool
	Threads int
	Seed    int64
	Cutoff  int
	Limit   time.Duration
	Perturb float64
}

type Stats struct {
	Players [2]struct {
		Wins     int
		FlatWins int
		RoadWins int
	}
	White, Black int
	Ties         int
	Cutoff       int

	Games []Result
}

type gameSpec struct {
	c            *Config
	i            int
	white, black *ai.MinimaxConfig
	r            *rand.Rand
	p1color      tak.Color
}

type Result struct {
	spec     gameSpec
	Initial  *tak.Position
	Position *tak.Position
	Moves    []tak.Move
}

func Simulate(c *Config) Stats {
	var st Stats
	rc := make(chan Result)
	go startGames(c, rc)
	for r := range rc {
		d := r.Position.WinDetails()
		if c.Verbose {
			log.Printf("game n=%d plies=%d p1=%s winner=%s wf=%d bf=%d ws=%d bs=%d",
				r.spec.i, r.Position.MoveNumber(),
				r.spec.p1color, d.Winner,
				d.WhiteFlats,
				d.BlackFlats,
				r.Position.WhiteStones(),
				r.Position.BlackStones(),
			)
		}
		if d.Over {
			if d.Winner == tak.White {
				st.White++
			} else if d.Winner == tak.Black {
				st.Black++
			} else {
				st.Ties++
			}
		} else {
			st.Cutoff++
		}
		if d.Over && d.Winner != tak.NoColor {
			pst := &st.Players[0]
			if d.Winner == r.spec.p1color.Flip() {
				pst = &st.Players[1]
			}
			pst.Wins++
			switch d.Reason {
			case tak.FlatsWin:
				pst.FlatWins++
			case tak.RoadWin:
				pst.RoadWins++
			}
		}
		st.Games = append(st.Games, r)
	}

	return st
}

func startGames(c *Config, rc chan<- Result) {
	gc := make(chan gameSpec)
	var wg sync.WaitGroup
	wg.Add(c.Threads)
	for i := 0; i < c.Threads; i++ {
		go func() {
			worker(gc, rc)
			wg.Done()
		}()
	}
	r := rand.New(rand.NewSource(c.Seed))
	for g := 0; g < c.Games; g++ {
		var white, black *ai.MinimaxConfig
		w1 := c.W1
		w2 := c.W2
		if c.Perturb != 0.0 {
			w1 = perturbWeights(c.Perturb, w1)
			w2 = perturbWeights(c.Perturb, w2)
		}
		cfg1 := c.Cfg1
		cfg1.Evaluate = ai.MakeEvaluator(c.Cfg1.Size, &w1)
		cfg1.Seed = r.Int63()

		cfg2 := c.Cfg2
		cfg2.Evaluate = ai.MakeEvaluator(c.Cfg1.Size, &w2)
		cfg2.Seed = r.Int63()

		var p1color tak.Color
		if g%2 == 0 || !c.Swap {
			white, black = &cfg1, &cfg2
			p1color = tak.White
		} else {
			black, white = &cfg1, &cfg2
			p1color = tak.Black
		}

		spec := gameSpec{
			c:       c,
			i:       g,
			white:   white,
			black:   black,
			p1color: p1color,
			r:       rand.New(rand.NewSource(r.Int63())),
		}
		gc <- spec
	}
	close(gc)
	wg.Wait()
	close(rc)
}

func worker(games <-chan gameSpec, out chan<- Result) {
	for g := range games {
		white := ai.NewMinimax(*g.white)
		black := ai.NewMinimax(*g.black)
		var ms []tak.Move
		var initial *tak.Position
		if len(g.c.Initial) != 0 {
			initial = g.c.Initial[g.r.Intn(len(g.c.Initial))]
		}
		p := initial
		if p == nil {
			p = tak.New(tak.Config{Size: g.c.Cfg1.Size})
		}
		for i := 0; i < g.c.Cutoff; i++ {
			var m tak.Move
			var cancel context.CancelFunc
			ctx := context.Background()
			if g.c.Limit != 0 {
				ctx, cancel = context.WithTimeout(ctx, g.c.Limit)
			}
			if p.ToMove() == tak.White {
				m = white.GetMove(ctx, p)
			} else {
				m = black.GetMove(ctx, p)
			}
			if cancel != nil {
				cancel()
			}
			var e error
			p, e = p.Move(&m)
			if e != nil {
				panic(fmt.Sprintf("illegal move: %s", ptn.FormatMove(&m)))
			}
			ms = append(ms, m)
			if ok, _ := p.GameOver(); ok {
				break
			}
		}
		out <- Result{
			spec:     g,
			Initial:  initial,
			Position: p,
			Moves:    ms,
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

	return w
}
