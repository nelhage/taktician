package selfplay

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

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

	Swap        bool
	Threads     int
	Seed        int64
	Cutoff      int
	Limit       time.Duration
	TimeControl time.Duration

	Perturb float64
}

type Stats struct {
	Players [2]struct {
		Wins      int
		WhiteWins int
		BlackWins int
		FlatWins  int
		RoadWins  int
		TimeWins  int
	}
	White, Black int
	Ties         int
	Cutoff       int

	Games []Result `json:"-"`
}

func (s *Stats) Count() int {
	return s.White + s.Black + s.Ties + s.Cutoff
}

func (s *Stats) Merge(other *Stats) Stats {
	out := *s
	for i := range out.Players {
		out.Players[i].Wins += other.Players[i].Wins
		out.Players[i].WhiteWins += other.Players[i].WhiteWins
		out.Players[i].BlackWins += other.Players[i].BlackWins
		out.Players[i].FlatWins += other.Players[i].FlatWins
		out.Players[i].RoadWins += other.Players[i].RoadWins
		out.Players[i].TimeWins += other.Players[i].TimeWins
	}
	out.White += other.White
	out.Black += other.Black
	out.Ties += other.Ties
	out.Cutoff += other.Cutoff
	return out
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
	Winner   tak.Color
}

func Simulate(c *Config) Stats {
	var st Stats
	rc := make(chan Result)
	go startGames(c, rc)
	for r := range rc {
		if c.Verbose {
			log.Printf("game n=%d/%d plies=%d p1=%s winner=%s ws=%d bs=%d",
				r.spec.oi, r.spec.i, r.Position.MoveNumber(),
				r.spec.p1color,
				r.Winner,
				r.Position.WhiteStones(),
				r.Position.BlackStones(),
			)
		}
		if r.Winner == tak.White {
			st.White++
		} else if r.Winner == tak.Black {
			st.Black++
		} else if over, _ := r.Position.GameOver(); over {
			st.Ties++
		} else {
			st.Cutoff++
		}
		if r.Winner != tak.NoColor {
			pst := &st.Players[0]
			if r.Winner == r.spec.p1color.Flip() {
				pst = &st.Players[1]
			}
			if r.Winner == tak.White {
				pst.WhiteWins += 1
			} else if r.Winner == tak.Black {
				pst.BlackWins += 1
			}
			pst.Wins++
			d := r.Position.WinDetails()
			if d.Over {
				switch d.Reason {
				case tak.FlatsWin:
					pst.FlatWins++
				case tak.RoadWin:
					pst.RoadWins++
				}
			} else {
				pst.TimeWins++
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
		var white, black *tei.Player

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
		var tc *tei.TimeControl
		if c.TimeControl != 0 {
			tc = &tei.TimeControl{
				White: c.TimeControl,
				Black: c.TimeControl,
			}
		}
		var winner tak.Color
		for i := 0; i < g.c.Cutoff; i++ {
			var m tak.Move
			var cancel context.CancelFunc
			ctx := context.Background()
			if g.c.Limit != 0 {
				ctx, cancel = context.WithTimeout(ctx, g.c.Limit)
			}
			var err error
			before := time.Now()
			if p.ToMove() == tak.White {
				m, err = white.TEIGetMove(ctx, p, tc)
			} else {
				m, err = black.TEIGetMove(ctx, p, tc)
			}
			duration := time.Since(before)
			if cancel != nil {
				cancel()
			}
			if tc != nil {
				var tm *time.Duration
				if p.ToMove() == tak.White {
					tm = &tc.White
				} else {
					tm = &tc.Black
				}
				*tm = *tm - duration
				if *tm <= time.Millisecond {
					winner = p.ToMove().Flip()
					break
				}
			}

			if err != nil {
				log.Fatalf("Get move: %s", err.Error())
			}
			var e error
			p, e = p.Move(m)
			if e != nil {
				panic(fmt.Sprintf("illegal move: %s", ptn.FormatMove(m)))
			}
			ms = append(ms, m)
			if ok, w := p.GameOver(); ok {
				winner = w
				break
			}
		}
		out <- Result{
			spec:     g,
			Initial:  g.opening,
			Position: p,
			Moves:    ms,
			Winner:   winner,
		}
	}
}
