package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"context"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/playtak"
	"github.com/nelhage/taktician/playtak/bot"
	"github.com/nelhage/taktician/tak"
)

const (
	minThink = 5 * time.Second
	maxThink = time.Minute

	undoTimeout = 30 * time.Second

	defaultLevel = 6

	docURL = "http://bit.ly/25h33rC"
)

type Friendly struct {
	client *playtak.Commands
	ai     ai.TakPlayer
	check  *ai.MinimaxAI
	g      *bot.Game

	fpa FPARule

	level    int
	levelSet time.Time

	log *os.File
}

func (f *Friendly) NewGame(g *bot.Game) {
	if time.Now().Sub(f.levelSet) > 1*time.Hour {
		f.level = defaultLevel
	}
	f.g = g
	f.ai = wrapWithBook(g.Size, ai.NewMinimax(f.AIConfig()))
	f.check = ai.NewMinimax(ai.MinimaxConfig{
		Depth:    3,
		Size:     g.Size,
		Debug:    0,
		TableMem: -1,
		Evaluate: ai.EvaluateWinner,
	})
	f.client.Tell(g.Opponent,
		fmt.Sprintf("%s@level %d: %s",
			*user, f.level, docURL))
	if f.fpa != nil {
		if gs := f.fpa.Greeting(g.Color); gs != nil {
			for _, m := range gs {
				f.client.Tell(g.Opponent, m)
			}
		}
	}
}

func (f *Friendly) GameOver() {
	if f.fpa != nil && f.g.Result != "" && f.g.Result != "0-1" && f.g.Result != "1-0" {
		url := f.fpa.SurveyURL()
		if url != "" {
			f.client.Tell(f.g.Opponent,
				fmt.Sprintf("Thanks for playing! Please share your feedback about this rule variation: %s", url))
		}
	}
	if *logFile != "" {
		l, e := os.OpenFile(*logFile,
			os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if e != nil {
			log.Printf("log: open(%s): %v", *logFile, e)
		} else {
			defer l.Close()
			var winner tak.Color
			var moves int
			if len(f.g.Positions) > 0 {
				p := f.g.Positions[len(f.g.Positions)-1]
				_, winner = p.GameOver()
				moves = p.MoveNumber()
			}
			fmt.Fprintf(l,
				"%s\t%s\t%s\t%d\t%s\t%d\t%s\t%s\t%d\n",
				f.g.ID,
				time.Now().Format(time.RFC3339),
				f.g.Color,
				f.g.Size,
				f.g.Opponent,
				f.level,
				f.g.Result,
				winner,
				moves,
			)
		}
	}
	f.g = nil
}

func (f *Friendly) GetMove(
	ctx context.Context,
	p *tak.Position,
	mine, theirs time.Duration) tak.Move {
	if f.fpa != nil {
		if p.MoveNumber() > 0 {
			prevP := f.g.Positions[len(f.g.Positions)-2]
			prevM := f.g.Moves[len(f.g.Moves)-1]
			if err := f.fpa.LegalMove(prevP, prevM); err != nil {
				f.client.SendCommand(f.g.GameStr, "Resign")
				f.client.Tell(f.g.Opponent, err.Error())
				<-ctx.Done()
				return tak.Move{}
			}
		}
	}

	if p.ToMove() != f.g.Color {
		return tak.Move{}
	}

	if f.fpa != nil {
		m, ok := f.fpa.GetMove(p)
		if ok {
			return m
		}
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

func (f *Friendly) Config(size int) tak.Config {
	cfg := tak.Config{Size: size}
	if f.fpa != nil {
		cfg.BlackWinsTies = true
	}
	return cfg
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
			f.ai = wrapWithBook(f.g.Size, ai.NewMinimax(f.AIConfig()))
			return fmt.Sprintf("OK! I'll play at level %d, starting right now.", l)
		}
	case "size":
		sz, err := strconv.Atoi(arg)
		if err != nil {
			log.Printf("bad size size=%q", arg)
			return ""
		}
		if sz >= 3 && sz <= 8 {
			*size = sz
			f.client.SendCommand("Seek",
				strconv.Itoa(*size),
				strconv.Itoa(int(gameTime.Seconds())),
				strconv.Itoa(int(increment.Seconds())))
		}
	case "help":
		return fmt.Sprintf("[%s@level %d]: %s",
			*user, f.level, docURL)
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

func (f *Friendly) AIConfig() ai.MinimaxConfig {
	cfg := ai.MinimaxConfig{
		Size:  f.g.Size,
		Debug: *debug,

		NoSort:   !*sort,
		TableMem: *tableMem,
		MultiCut: *multicut,
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
		Groups:           [8]int{0, 0, 0, 100, 200, 300, 310, 320},
	}
)

func constw(w ai.Weights) func(int) *ai.Weights {
	return func(int) *ai.Weights { return &w }
}

func indexw(ws []ai.Weights) func(int) *ai.Weights {
	return func(sz int) *ai.Weights {
		if sz < len(ws) {
			return &ws[sz]
		}
		panic("bad weights/size")
	}
}

var levels = []struct {
	depth   int
	weights func(size int) *ai.Weights
}{
	{2, constw(easyWeights)},
	{2, constw(medWeights)},
	{2, indexw(ai.DefaultWeights)},
	{3, constw(easyWeights)},
	{3, constw(medWeights)},
	{4, constw(medWeights)},
	{3, indexw(ai.DefaultWeights)},
	{5, constw(easyWeights)},
	{5, constw(medWeights)},
	{4, indexw(ai.DefaultWeights)},
	{5, indexw(ai.DefaultWeights)},
	{7, indexw(ai.DefaultWeights)},
	{0, indexw(ai.DefaultWeights)},
}

func (f *Friendly) levelSettings(size int, level int) (int, ai.EvaluationFunc) {
	if level == 0 {
		level = 3
	}
	if level > len(levels) {
		level = len(levels)
	}
	s := levels[level-1]
	return s.depth, ai.MakeEvaluator(size, s.weights(size))
}

func (f *Friendly) AcceptUndo() bool {
	return true
}
