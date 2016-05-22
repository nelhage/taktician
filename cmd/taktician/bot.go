package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/nelhage/taktician/playtak"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

type Game struct {
	id       string
	opponent string
	color    tak.Color
	size     int
	time     time.Duration
}

type Bot interface {
	NewGame(g *Game)
	GetMove(p *tak.Position, mine, theirs time.Duration) tak.Move
	HandleChat(who, msg string)
}

func playGame(c *playtak.Client, b Bot, line string) {
	var g Game
	bits := strings.Split(line, " ")
	g.size, _ = strconv.Atoi(bits[3])
	g.id = bits[2]
	switch bits[7] {
	case "white":
		g.color = tak.White
		g.opponent = bits[6]
	case "black":
		g.color = tak.Black
		g.opponent = bits[4]
	default:
		panic(fmt.Sprintf("bad color: %s", bits[7]))
	}

	secs, _ := strconv.Atoi(bits[8])
	g.time = time.Duration(secs) * time.Second

	gameStr := fmt.Sprintf("Game#%s", g.id)
	p := tak.New(tak.Config{Size: g.size})
	b.NewGame(&g)

	moves := make(chan tak.Move, 1)

	log.Printf("new game game-id=%q size=%d opponent=%q color=%q time=%q",
		g.id, g.size, g.opponent, g.color, g.time)

	var times struct {
		mine, theirs time.Duration
	}
	times.mine = g.time
	times.theirs = g.time

	for {
		over, _ := p.GameOver()
		if g.color == p.ToMove() && !over {
			go func() {
				moves <- b.GetMove(p, times.mine, times.theirs)
			}()
		}

		var timeout <-chan time.Time
	eventLoop:
		for {
			var line string
			select {
			case line = <-c.Recv:
			case move := <-moves:
				next, err := p.Move(&move)
				if err != nil {
					log.Printf("ai returned bad move: %s: %s",
						ptn.FormatMove(&move), err)
					break eventLoop
				}
				p = next
				c.SendCommand(gameStr, playtak.FormatServer(&move))
				log.Printf("my-move game-id=%s ply=%d ptn=%d.%s move=%q",
					g.id,
					p.MoveNumber(),
					p.MoveNumber()/2+1,
					strings.ToUpper(g.color.String()[:1]),
					ptn.FormatMove(&move))
			case <-timeout:
				break eventLoop
			}

			bits = strings.Split(line, " ")
			switch bits[0] {
			case gameStr:
			case "Shout":
				who, msg := playtak.ParseShout(line)
				if who != "" {
					b.HandleChat(who, msg)
				}
				fallthrough
			default:
				continue eventLoop
			}
			switch bits[1] {
			case "P", "M":
				move, err := playtak.ParseServer(strings.Join(bits[1:], " "))
				if err != nil {
					panic(err)
				}
				p, err = p.Move(&move)
				if err != nil {
					panic(err)
				}
				log.Printf("their-move game-id=%s ply=%d ptn=%d.%s move=%q",
					g.id,
					p.MoveNumber(),
					p.MoveNumber()/2+1,
					strings.ToUpper(g.color.Flip().String()[:1]),
					ptn.FormatMove(&move))
				timeout = time.NewTimer(500 * time.Millisecond).C
			case "Abandoned.":
				log.Printf("game-over game-id=%s ply=%d result=abandoned",
					g.id, p.MoveNumber())
				return
			case "Over":
				log.Printf("game-over game-id=%s ply=%d result=%q",
					g.id, p.MoveNumber(), bits[2])
				return
			case "Time":
				w, _ := strconv.Atoi(bits[2])
				b, _ := strconv.Atoi(bits[3])
				if g.color == tak.White {
					times.mine = time.Duration(w) * time.Second
					times.theirs = time.Duration(b) * time.Second
				} else {
					times.theirs = time.Duration(w) * time.Second
					times.mine = time.Duration(b) * time.Second
				}
				break eventLoop
			}
		}
	}
}
