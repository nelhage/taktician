package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/nelhage/taktician/playtak"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

type Game struct {
	p        *tak.Position
	id       string
	gameStr  string
	opponent string
	color    tak.Color
	size     int
	time     time.Duration

	times struct {
		mine, theirs time.Duration
	}

	bot      Bot
	moveLock sync.Mutex

	positions []*tak.Position
	moves     []tak.Move
}

type Bot interface {
	NewGame(g *Game)
	GameOver()
	GetMove(ctx context.Context,
		p *tak.Position,
		mine, theirs time.Duration) tak.Move
	AcceptUndo() bool
	HandleChat(who, msg string)
}

type Client interface {
	Recv() <-chan string
	SendCommand(...string)
}

func parseGameStart(line string) *Game {
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
	return &g
}

func playGame(c Client, b Bot, line string) {
	ctx := context.Background()
	g := parseGameStart(line)

	g.gameStr = fmt.Sprintf("Game#%s", g.id)
	g.p = tak.New(tak.Config{Size: g.size})
	g.positions = append(g.positions, g.p)
	g.bot = b
	b.NewGame(g)
	defer b.GameOver()

	log.Printf("new game game-id=%q size=%d opponent=%q color=%q time=%q",
		g.id, g.size, g.opponent, g.color, g.time)

	g.times.mine = g.time
	g.times.theirs = g.time

	for {
		over, _ := g.p.GameOver()
		if over {
			break
		}
		if handleMove(ctx, g, c) {
			break
		}
	}
}

func handleMove(ctx context.Context, g *Game, c Client) bool {
	moves := make(chan tak.Move, 1)
	moveCtx, moveCancel := context.WithCancel(ctx)
	defer moveCancel()
	go func(p *tak.Position, mc chan<- tak.Move) {
		g.moveLock.Lock()
		defer g.moveLock.Unlock()
		defer moveCancel()
		mc <- g.bot.GetMove(moveCtx, p, g.times.mine, g.times.theirs)
	}(g.p, moves)
	if g.p.ToMove() != g.color {
		moves = nil
	}

	var timeout <-chan time.Time

	for {
		var line string
		var ok bool
		select {
		case line, ok = <-c.Recv():
			if !ok {
				return false
			}
		case move := <-moves:
			next, err := g.p.Move(&move)
			if err != nil {
				log.Printf("ai returned bad move: %s: %s",
					ptn.FormatMove(&move), err)
				return false
			}
			c.SendCommand(g.gameStr, playtak.FormatServer(&move))
			log.Printf("my-move game-id=%s ply=%d ptn=%d.%s move=%q",
				g.id,
				g.p.MoveNumber(),
				g.p.MoveNumber()/2+1,
				strings.ToUpper(g.p.ToMove().String()[:1]),
				ptn.FormatMove(&move))
			g.p = next
			g.positions = append(g.positions, g.p)
			g.moves = append(g.moves, move)
			return false
		case <-timeout:
			return false
		}

		bits := strings.Split(line, " ")
		switch bits[0] {
		case g.gameStr:
		case "Shout":
			who, msg := playtak.ParseShout(line)
			if who != "" {
				g.bot.HandleChat(who, msg)
			}
			fallthrough
		default:
			continue
		}
		switch bits[1] {
		case "P", "M":
			move, err := playtak.ParseServer(strings.Join(bits[1:], " "))
			if err != nil {
				panic(err)
			}
			next, err := g.p.Move(&move)
			if err != nil {
				panic(err)
			}
			log.Printf("their-move game-id=%s ply=%d ptn=%d.%s move=%q",
				g.id,
				g.p.MoveNumber(),
				g.p.MoveNumber()/2+1,
				strings.ToUpper(g.p.ToMove().String()[:1]),
				ptn.FormatMove(&move))
			g.p = next
			g.positions = append(g.positions, g.p)
			g.moves = append(g.moves, move)
			timeout = time.After(500 * time.Millisecond)
		case "Abandoned.":
			log.Printf("game-over game-id=%s opponent=%s ply=%d result=abandoned",
				g.id, g.opponent, g.p.MoveNumber())
			return true
		case "Over":
			log.Printf("game-over game-id=%s opponent=%s ply=%d result=%q",
				g.id, g.opponent, g.p.MoveNumber(), bits[2])
			return true
		case "Time":
			w, _ := strconv.Atoi(bits[2])
			b, _ := strconv.Atoi(bits[3])
			if g.color == tak.White {
				g.times.mine = time.Duration(w) * time.Second
				g.times.theirs = time.Duration(b) * time.Second
			} else {
				g.times.theirs = time.Duration(w) * time.Second
				g.times.mine = time.Duration(b) * time.Second
			}
			return false
		case "RequestUndo":
			if g.bot.AcceptUndo() {
				c.SendCommand(g.gameStr, "RequestUndo")
				moveCancel()
			}
		case "Undo":
			log.Printf("undo game-id=%s ply=%d", g.id, g.p.MoveNumber())
			g.positions = g.positions[:len(g.positions)-1]
			g.moves = g.moves[:len(g.moves)-1]
			g.p = g.positions[len(g.positions)-1]
			return false
		}
	}
}
