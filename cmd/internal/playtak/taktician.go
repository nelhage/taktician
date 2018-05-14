package playtak

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/playtak"
	"github.com/nelhage/taktician/playtak/bot"
	"github.com/nelhage/taktician/tak"
)

type Taktician struct {
	cmd *Command

	g      *bot.Game
	client *playtak.Commands
	ai     ai.TakPlayer
}

func (t *Taktician) NewGame(g *bot.Game) {
	t.g = g
	t.ai = t.cmd.wrapWithBook(
		g.Size,
		ai.NewMinimax(ai.MinimaxConfig{
			Size:  g.Size,
			Depth: t.cmd.depth,
			Debug: t.cmd.debug,

			NoSort:   !t.cmd.sort,
			TableMem: t.cmd.tableMem,
			MultiCut: t.cmd.multicut,
		}))
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
	} else if !t.cmd.useOpponentTime {
		return tak.Move{}
	}
	return t.ai.GetMove(ctx, p)
}

func (t *Taktician) timeBound(remaining time.Duration) time.Duration {
	if t.g.Size == 4 {
		return t.cmd.limit
	}
	return t.cmd.limit
}

func (t *Taktician) GameOver() {
	t.ai = nil
}

func (t *Taktician) handleCommand(cmd, arg string) {
	switch strings.ToLower(cmd) {
	case "size":
		sz, err := strconv.Atoi(arg)
		if err != nil {
			log.Printf("bad size size=%q", arg)
			return
		}
		if sz >= 4 && sz <= 6 {
			t.cmd.size = sz
			t.client.SendCommand("Seek",
				strconv.Itoa(t.cmd.size),
				strconv.Itoa(int(t.cmd.gameTime.Seconds())),
				strconv.Itoa(int(t.cmd.increment.Seconds())))
		}
	}
}

func (t *Taktician) HandleTell(who string, msg string) {
	bits := strings.SplitN(msg, " ", 2)
	cmd := bits[0]
	var arg string
	if len(bits) == 2 {
		arg = bits[1]
	}
	t.handleCommand(cmd, arg)
}

func (t *Taktician) HandleChat(room string, who string, msg string) {
	cmd, arg := parseCommand(t.client.User, msg)
	if cmd == "" {
		return
	}
	log.Printf("chat room=%q from=%q msg=%q", room, who, msg)
	t.handleCommand(cmd, arg)
}

func (t *Taktician) AcceptUndo() bool {
	return false
}
