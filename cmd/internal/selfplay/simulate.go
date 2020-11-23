package selfplay

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
	"github.com/nelhage/taktician/tei"
)

type Config struct {
	Games int

	Verbose bool

	Initial []*tak.Position

	P1, P2 []string

	Zero  bool
	Size  int
	Depth int
	Debug int

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
	c       *Config
	opening *tak.Position
	oi      int
	i       int
	r       *rand.Rand
	p1color tak.Color
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
				r.spec.p1color,
				d.Winner,
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
			worker(c, gc, rc)
			wg.Done()
		}()
	}
	r := rand.New(rand.NewSource(c.Seed))
	for pi, pos := range c.Initial {
		n := c.Games
		if c.Swap {
			n *= 2
		}
		for g := 0; g < n; g++ {
			var p1color tak.Color
			if g%2 == 0 || !c.Swap {
				p1color = tak.White
			} else {
				p1color = tak.Black
			}

			spec := gameSpec{
				opening: pos,
				c:       c,
				oi:      pi,
				i:       g,
				p1color: p1color,
				r:       rand.New(rand.NewSource(r.Int63())),
			}
			gc <- spec
		}
	}
	close(gc)
	wg.Wait()
	close(rc)
}

func worker(c *Config, games <-chan gameSpec, out chan<- Result) {
	c1, err := tei.NewClient(c.P1)
	if err != nil {
		log.Fatalf("starting client[%v]: %v", c.P1, err)
	}
	defer c1.Close()
	c2, err := tei.NewClient(c.P2)
	if err != nil {
		log.Fatalf("starting client[%v]: %v", c.P2, err)
	}
	defer c2.Close()

	for g := range games {
		var white, black ai.TakPlayer

		white, err = c1.NewGame(g.opening.Size())
		if err != nil {
			log.Fatalf("starting game[%v]: %v", c.P1, err)
		}
		black, err = c2.NewGame(g.opening.Size())
		if err != nil {
			log.Fatalf("starting game[%v]: %v", c.P2, err)
		}
		if g.p1color != tak.White {
			white, black = black, white
		}

		var ms []tak.Move
		p := g.opening
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
			p, e = p.Move(m)
			if e != nil {
				panic(fmt.Sprintf("illegal move: %s", ptn.FormatMove(m)))
			}
			ms = append(ms, m)
			if ok, _ := p.GameOver(); ok {
				break
			}
		}
		out <- Result{
			spec:     g,
			Initial:  g.opening,
			Position: p,
			Moves:    ms,
		}
	}
}
