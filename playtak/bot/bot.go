package bot

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
	ID       string
	GameStr  string
	Opponent string
	Color    tak.Color
	Size     int
	Time     time.Duration

	times struct {
		mine, theirs time.Duration
	}

	p *tak.Position

	bot      Bot
	moveLock sync.Mutex

	Positions []*tak.Position
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
	g.Size, _ = strconv.Atoi(bits[3])
	g.ID = bits[2]
	switch bits[7] {
	case "white":
		g.Color = tak.White
		g.Opponent = bits[6]
	case "black":
		g.Color = tak.Black
		g.Opponent = bits[4]
	default:
		panic(fmt.Sprintf("bad color: %s", bits[7]))
	}

	secs, _ := strconv.Atoi(bits[8])
	g.Time = time.Duration(secs) * time.Second
	return &g
}

func PlayGame(c Client, b Bot, line string) {
	ctx := context.Background()
	g := parseGameStart(line)

	g.GameStr = fmt.Sprintf("Game#%s", g.ID)
	g.p = tak.New(tak.Config{Size: g.Size})
	g.Positions = append(g.Positions, g.p)
	g.bot = b
	b.NewGame(g)
	defer b.GameOver()

	log.Printf("new game game-id=%q size=%d opponent=%q color=%q time=%q",
		g.ID, g.Size, g.Opponent, g.Color, g.Time)

	g.times.mine = g.Time
	g.times.theirs = g.Time

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
	if g.p.ToMove() != g.Color {
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
			c.SendCommand(g.GameStr, playtak.FormatServer(&move))
			log.Printf("my-move game-id=%s ply=%d ptn=%d.%s move=%q",
				g.ID,
				g.p.MoveNumber(),
				g.p.MoveNumber()/2+1,
				strings.ToUpper(g.p.ToMove().String()[:1]),
				ptn.FormatMove(&move))
			g.p = next
			g.Positions = append(g.Positions, g.p)
			g.moves = append(g.moves, move)
			return false
		case <-timeout:
			return false
		}

		bits := strings.Split(line, " ")
		switch bits[0] {
		case g.GameStr:
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
				g.ID,
				g.p.MoveNumber(),
				g.p.MoveNumber()/2+1,
				strings.ToUpper(g.p.ToMove().String()[:1]),
				ptn.FormatMove(&move))
			g.p = next
			g.Positions = append(g.Positions, g.p)
			g.moves = append(g.moves, move)
			timeout = time.After(500 * time.Millisecond)
		case "Abandoned.":
			log.Printf("game-over game-id=%s opponent=%s ply=%d result=abandoned",
				g.ID, g.Opponent, g.p.MoveNumber())
			return true
		case "Over":
			log.Printf("game-over game-id=%s opponent=%s ply=%d result=%q",
				g.ID, g.Opponent, g.p.MoveNumber(), bits[2])
			return true
		case "Time":
			w, _ := strconv.Atoi(bits[2])
			b, _ := strconv.Atoi(bits[3])
			if g.Color == tak.White {
				g.times.mine = time.Duration(w) * time.Second
				g.times.theirs = time.Duration(b) * time.Second
			} else {
				g.times.theirs = time.Duration(w) * time.Second
				g.times.mine = time.Duration(b) * time.Second
			}
			return false
		case "RequestUndo":
			if g.bot.AcceptUndo() {
				c.SendCommand(g.GameStr, "RequestUndo")
				moveCancel()
				moves = nil
			}
		case "Undo":
			log.Printf("undo game-id=%s ply=%d", g.ID, g.p.MoveNumber())
			g.Positions = g.Positions[:len(g.Positions)-1]
			g.moves = g.moves[:len(g.moves)-1]
			g.p = g.Positions[len(g.Positions)-1]
			return false
		}
	}
}
