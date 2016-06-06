package main

import (
	"log"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/playtak"
	"github.com/nelhage/taktician/tak"
)

func timeBound(remaining time.Duration) time.Duration {
	return *limit
}

type Taktician struct {
	client *playtak.Client
	ai     *ai.MinimaxAI
}

func (t *Taktician) NewGame(g *Game) {
	t.ai = ai.NewMinimax(ai.MinimaxConfig{
		Size:  g.size,
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
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(timeBound(mine)))
	defer cancel()
	return t.ai.GetMove(ctx, p)
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
