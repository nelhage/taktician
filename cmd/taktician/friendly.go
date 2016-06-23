package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/playtak"
	"github.com/nelhage/taktician/playtak/bot"
	"github.com/nelhage/taktician/tak"
)

const (
	minThink = 5 * time.Second
	maxThink = time.Minute

	undoTimeout = 30 * time.Second

	defaultLevel = 2

	docURL = "http://bit.ly/25h33rC"
)

type Friendly struct {
	client *playtak.Client
	ai     *ai.MinimaxAI
	check  *ai.MinimaxAI
	g      *bot.Game

	level    int
	levelSet time.Time

	greeted map[string]time.Time
}

func (f *Friendly) NewGame(g *bot.Game) {
	if f.greeted == nil {
		f.greeted = make(map[string]time.Time)
	}
	if time.Now().Sub(f.levelSet) > 1*time.Hour {
		f.level = defaultLevel
	}
	f.g = g
	f.ai = ai.NewMinimax(f.Config())
	f.check = ai.NewMinimax(ai.MinimaxConfig{
		Depth:    3,
		Size:     g.Size,
		Debug:    0,
		Evaluate: ai.EvaluateWinner,
	})
	/*
		f.client.SendCommand("Shout",
			fmt.Sprintf("[FriendlyBot@level %d] Good luck %s!",
				f.level, g.opponent,
			))
	*/
	if t := f.greeted[g.Opponent]; time.Now().Sub(t) > time.Hour {
		log.Printf("greeting user=%q greeted=%s", g.Opponent, t)
		f.client.SendCommand("Shout",
			fmt.Sprintf("FriendlyBot@level %d: %s",
				f.level, docURL))
		f.greeted[g.Opponent] = time.Now()
	}
}

func (f *Friendly) GameOver() {
	f.g = nil
}

func (f *Friendly) GetMove(
	ctx context.Context,
	p *tak.Position,
	mine, theirs time.Duration) tak.Move {
	if p.ToMove() != f.g.Color {
		return tak.Move{}
	}
	var deadline <-chan time.Time
	if f.waitUndo(p) {
		deadline = time.After(undoTimeout)
	} else {
		deadline = time.After(minThink)
	}
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(maxThink))
	defer cancel()
	m := f.ai.GetMove(ctx, p)
	select {
	case <-deadline:
	case <-ctx.Done():
	}

	return m
}

func (f *Friendly) waitUndo(p *tak.Position) bool {
	ctx := context.Background()
	_, v, st := f.check.Analyze(ctx, p)
	if v < ai.WinThreshold || st.Depth > 1 {
		return false
	}
	_, v, st = f.check.Analyze(ctx, f.g.Positions[len(f.g.Positions)-2])
	if v > -ai.WinThreshold {
		return true
	}
	return false
}

func (f *Friendly) HandleChat(who string, msg string) {
	log.Printf("chat from=%q msg=%q", who, msg)
	cmd, arg := parseCommand(msg)
	if cmd == "" {
		return
	}

	switch cmd {
	case "level":
		if arg == "max" {
			f.level = 100
			f.levelSet = time.Now()
			f.client.SendCommand("Shout", "OK! I'll play as best as I can!")
			break
		}
		l, e := strconv.ParseUint(arg, 10, 64)
		if e != nil {
			log.Printf("bad level: %v", e)
			return
		}
		if int(l) < 1 || int(l) > len(levels)+1 {
			f.client.SendCommand("Shout", fmt.Sprintf("I only know about levels up to %d", len(levels)+1))
			break
		}
		f.level = int(l)
		f.levelSet = time.Now()
		if f.g == nil || who != f.g.Opponent {
			f.client.SendCommand("Shout",
				fmt.Sprintf("OK! I'll play at level %d for future games.", l))
		} else if f.g != nil {
			f.ai = ai.NewMinimax(f.Config())
			f.client.SendCommand("Shout",
				fmt.Sprintf("OK! I'll play at level %d, starting right now.", l))
		}
	case "size":
		sz, err := strconv.Atoi(arg)
		if err != nil {
			log.Printf("bad size size=%q", arg)
			return
		}
		if sz >= 4 && sz <= 6 {
			*size = sz
			f.client.SendCommand("Seek",
				strconv.Itoa(*size),
				strconv.Itoa(int(gameTime.Seconds())),
				strconv.Itoa(int(increment.Seconds())))
		}
	case "help":
		f.client.SendCommand("Shout",
			fmt.Sprintf("[FriendlyBot@level %d]: %s",
				f.level, docURL))
	}
}

func (f *Friendly) Config() ai.MinimaxConfig {
	cfg := ai.MinimaxConfig{
		Size:  f.g.Size,
		Debug: *debug,

		NoSort:  !*sort,
		NoTable: !*table,
	}
	cfg.Depth, cfg.Evaluate = f.levelSettings(f.g.Size, f.level)

	return cfg
}

var (
	easyWeights = ai.Weights{
		TopFlat: 100,
	}
	medWeights = ai.Weights{
		TopFlat:          200,
		Standing:         100,
		Capstone:         150,
		FlatCaptives:     ai.FlatScores{Hard: 50},
		StandingCaptives: ai.FlatScores{Hard: 50},
		CapstoneCaptives: ai.FlatScores{Hard: 50},
		Groups:           [8]int{0, 0, 0, 100, 200},
	}
)

var levels = []struct {
	depth   int
	weights ai.Weights
}{
	{2, easyWeights},
	{2, medWeights},
	{2, ai.DefaultWeights[5]},
	{3, easyWeights},
	{3, medWeights},
	{4, medWeights},
	{3, ai.DefaultWeights[5]},
	{5, easyWeights},
	{5, medWeights},
	{4, ai.DefaultWeights[5]},
	{5, ai.DefaultWeights[5]},
	{7, ai.DefaultWeights[5]},
}

func (f *Friendly) levelSettings(size int, level int) (int, ai.EvaluationFunc) {
	if level == 0 {
		level = 3
	}
	if level > len(levels) {
		level = len(levels)
	}
	s := levels[level-1]
	return s.depth, ai.MakeEvaluator(size, &s.weights)
}

func (f *Friendly) AcceptUndo() bool {
	return true
}
