package main

import (
	"log"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/playtak"
	"github.com/nelhage/taktician/playtak/bot"
	"github.com/nelhage/taktician/tak"
)

type Taktician struct {
	g      *bot.Game
	client *playtak.Client
	ai     *ai.MinimaxAI
}

func (t *Taktician) NewGame(g *bot.Game) {
	t.g = g
	t.ai = ai.NewMinimax(ai.MinimaxConfig{
		Size:  g.Size,
		Depth: *depth,
		Debug: *debug,

		NoSort:  !*sort,
		NoTable: !*table,
	})
}

func (t *Taktician) GetMove(
	ctx context.Context,
	p *tak.Position,
	mine, theirs time.Duration) tak.Move {
	if p.ToMove() == t.g.Color {
		var cancel context.CancelFunc
		timeout := t.timeBound(mine)
		if p.MoveNumber() < 2 {
			timeout = 20 * time.Second
		}
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	} else if !*useOpponentTime {
		return tak.Move{}
	}
	return t.ai.GetMove(ctx, p)
}

func (t *Taktician) timeBound(remaining time.Duration) time.Duration {
	if t.g.Size == 4 {
		return *limit
	}
	return *limit
}

func (t *Taktician) GameOver() {
	t.ai = nil
}

func (t *Taktician) HandleChat(who string, msg string) {
	cmd, arg := parseCommand(msg)
	if cmd == "" {
		return
	}
	log.Printf("chat from=%q msg=%q", who, msg)
	switch cmd {
	case "size":
		sz, err := strconv.Atoi(arg)
		if err != nil {
			log.Printf("bad size size=%q", arg)
			return
		}
		if sz >= 4 && sz <= 6 {
			*size = sz
			t.client.SendCommand("Seek",
				strconv.Itoa(*size),
				strconv.Itoa(int(gameTime.Seconds())),
				strconv.Itoa(int(increment.Seconds())))
		}
	}
}

func (t *Taktician) AcceptUndo() bool {
	return false
}
