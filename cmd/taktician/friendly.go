package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
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
}

func (f *Friendly) NewGame(g *bot.Game) {
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
	f.client.Tell(g.Opponent,
		fmt.Sprintf("FriendlyBot@level %d: %s",
			f.level, docURL))
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

func (f *Friendly) handleCommand(who, cmd, arg string) string {
	switch strings.ToLower(cmd) {
	case "level":
		if arg == "max" {
			f.level = 100
			f.levelSet = time.Now()
			return "OK! I'll play as best as I can!"
		}
		l, e := strconv.ParseUint(arg, 10, 64)
		if e != nil {
			log.Printf("bad level: %v", e)
			return ""
		}
		if int(l) < 1 || int(l) > len(levels)+1 {
			return fmt.Sprintf("I only know about levels up to %d", len(levels)+1)
		}
		f.level = int(l)
		f.levelSet = time.Now()
		if f.g == nil || who != f.g.Opponent {
			return fmt.Sprintf("OK! I'll play at level %d for future games.", l)
		} else if f.g != nil {
			f.ai = ai.NewMinimax(f.Config())
			return fmt.Sprintf("OK! I'll play at level %d, starting right now.", l)
		}
	case "size":
		sz, err := strconv.Atoi(arg)
		if err != nil {
			log.Printf("bad size size=%q", arg)
			return ""
		}
		if sz >= 4 && sz <= 6 {
			*size = sz
			f.client.SendCommand("Seek",
				strconv.Itoa(*size),
				strconv.Itoa(int(gameTime.Seconds())),
				strconv.Itoa(int(increment.Seconds())))
		}
	case "help":
		return fmt.Sprintf("[FriendlyBot@level %d]: %s",
			f.level, docURL)
	}
	return ""
}

func (f *Friendly) HandleTell(who string, msg string) {
	bits := strings.SplitN(msg, " ", 2)
	cmd := bits[0]
	var arg string
	if len(bits) == 2 {
		arg = bits[1]
	}

	if reply := f.handleCommand(who, cmd, arg); reply != "" {
		f.client.Tell(who, reply)
	}
}

func (f *Friendly) HandleChat(room string, who string, msg string) {
	log.Printf("chat room=%q from=%q msg=%q", room, who, msg)
	cmd, arg := parseCommand(msg)
	if cmd == "" {
		return
	}
	if reply := f.handleCommand(who, cmd, arg); reply != "" {
		f.client.Shout(room, reply)
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
	{0, ai.DefaultWeights[5]},
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
