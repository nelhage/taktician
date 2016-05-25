package main

import (
	"log"
	"strconv"
	"time"

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

func (t *Taktician) GetMove(p *tak.Position, mine, theirs time.Duration) tak.Move {
	return t.ai.GetMove(p, timeBound(mine))
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
		if sz == 5 || sz == 6 {
			*size = sz
			t.client.SendCommand("Seek",
				strconv.Itoa(*size),
				strconv.Itoa(int(gameTime.Seconds())),
				strconv.Itoa(int(increment.Seconds())))
		}
	}
}
