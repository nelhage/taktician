package main

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/playtak"
	"github.com/nelhage/taktician/tak"
)

const (
	minThink = 2 * time.Second
	maxThink = time.Minute

	defaultLevel = 2

	docURL = "https://github.com/nelhage/taktician/blob/master/doc/friendly.md"
)

var commandRE = regexp.MustCompile(`^([^ :]+):\s*([^ ]+)\s*(.*)$`)

type Friendly struct {
	client *playtak.Client
	ai     *ai.MinimaxAI
	g      *Game

	level    int
	levelSet time.Time

	greeted map[string]time.Time
}

func (f *Friendly) NewGame(g *Game) {
	if f.greeted == nil {
		f.greeted = make(map[string]time.Time)
	}
	if time.Now().Sub(f.levelSet) > 1*time.Hour {
		f.level = defaultLevel
	}
	f.g = g
	f.ai = ai.NewMinimax(f.Config())
	f.client.SendCommand("Shout",
		fmt.Sprintf("[FriendlyBot@level %d] Good luck %s!",
			f.level, g.opponent,
		))
	if g := f.greeted[g.opponent]; time.Now().Sub(g) > time.Hour {
		f.client.SendCommand("Shout",
			fmt.Sprintf("Learn more about me: %s",
				docURL))
	}
	f.greeted[g.opponent] = time.Now()
}

func (f *Friendly) GameOver() {
	f.g = nil
}

func (f *Friendly) GetMove(p *tak.Position, mine, theirs time.Duration) tak.Move {
	deadline := time.After(minThink)
	m := f.ai.GetMove(p, maxThink)
	<-deadline
	return m
}

func (f *Friendly) HandleChat(who string, msg string) {
	log.Printf("chat from=%q msg=%q", who, msg)
	gs := commandRE.FindStringSubmatch(msg)
	if gs == nil {
		return
	}
	if !strings.EqualFold(gs[1], *user) {
		return
	}
	switch gs[2] {
	case "level":
		if gs[3] == "max" {
			f.level = 100
			f.levelSet = time.Now()
			f.client.SendCommand("Shout", "OK! I'll play as best as I can!")
			break
		}
		l, e := strconv.ParseUint(strings.Trim(gs[3], " "), 10, 64)
		if e != nil {
			log.Printf("bad level: %v", e)
			return
		}
		if int(l) < 1 || int(l) > len(levels)+1 {
			f.client.SendCommand("Shout", fmt.Sprintf("I only know about levels up to %d", len(levels)+1))
		}
		f.level = int(l)
		f.levelSet = time.Now()
		if f.g == nil || who != f.g.opponent {
			f.client.SendCommand("Shout",
				fmt.Sprintf("OK! I'll play at level %d for future games.", l))
		} else if f.g != nil {
			f.ai = ai.NewMinimax(f.Config())
			f.client.SendCommand("Shout",
				fmt.Sprintf("OK! I'll play at level %d, starting right now.", l))
		}
	case "help":
		f.client.SendCommand("Shout",
			fmt.Sprintf("Learn more: %s", docURL))
	}
}

func (f *Friendly) Config() ai.MinimaxConfig {
	cfg := ai.MinimaxConfig{
		Size:  f.g.size,
		Debug: *debug,

		NoSort:  !*sort,
		NoTable: !*table,
	}
	cfg.Depth, cfg.Evaluate = f.levelSettings(f.g.size, f.level)

	return cfg
}

var (
	easyWeights = ai.Weights{
		TopFlat: 100,
		Tempo:   50,
	}
	medWeights = ai.Weights{
		TopFlat:  200,
		Standing: 100,
		Capstone: 150,
		Flat:     100,
		Tempo:    100,
		Groups:   [8]int{0, 0, 0, 100, 200},
	}
)

var levels = []struct {
	depth   int
	weights ai.Weights
}{
	{2, easyWeights},
	{2, medWeights},
	{3, easyWeights},
	{2, ai.DefaultWeights},
	{3, medWeights},
	{3, easyWeights},
	{4, medWeights},
	{3, ai.DefaultWeights},
	{5, easyWeights},
	{5, medWeights},
	{4, ai.DefaultWeights},
	{5, ai.DefaultWeights},
}

func (f *Friendly) levelSettings(size int, level int) (int, ai.EvaluationFunc) {
	if level == 0 {
		level = 3
	}
	if level > len(levels)+1 {
		return 7, ai.DefaultEvaluate
	}
	s := levels[level-1]
	return s.depth, ai.MakeEvaluator(&s.weights)
}
