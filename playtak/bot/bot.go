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
	HandleChat(room, who, msg string)
	HandleTell(who, msg string)
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

func parseObserveStart(line string) *Game {
	// Observe Game#58818 ShlktBot vs Gray_Mouser, 5x5, 1800, 0, 0 half-moves played, ShlktBot to move
	var g Game
	bits := strings.Split(line, " ")
	g.Size, _ = strconv.Atoi(bits[5][:1])
	g.GameStr = bits[1]
	g.ID = g.GameStr[len("Game#"):]
	g.Color = tak.NoColor
	secs, _ := strconv.Atoi(strings.TrimRight(bits[6], ","))
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

	for !handleMove(ctx, g, c) {
	}
}

func ObserveGame(c Client, b Bot, observe string) {
	ctx := context.Background()
	g := parseObserveStart(observe)

	g.p = tak.New(tak.Config{Size: g.Size})
	g.Positions = append(g.Positions, g.p)
	g.bot = b
	b.NewGame(g)
	defer b.GameOver()

	log.Printf("observe game-id=%q size=%d time=%q",
		g.ID, g.Size, g.Time)

	g.times.mine = g.Time
	g.times.theirs = g.Time

	for !handleMove(ctx, g, c) {
	}
}

func handleMove(ctx context.Context, g *Game, c Client) bool {
	moves := make(chan tak.Move, 1)
	moveCtx, moveCancel := context.WithCancel(ctx)
	defer moveCancel()
	go func(p *tak.Position, mc chan<- tak.Move, mine, theirs time.Duration) {
		if over, _ := p.GameOver(); over {
			<-moveCtx.Done()
			return
		}
		g.moveLock.Lock()
		defer g.moveLock.Unlock()
		defer moveCancel()
		mc <- g.bot.GetMove(moveCtx, p, mine, theirs)
	}(g.p, moves, g.times.mine, g.times.theirs)
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
				return true
			}
		case move := <-moves:
			next, err := g.p.Move(move)
			if err != nil {
				log.Printf("ai returned bad move: %s: %s",
					ptn.FormatMove(move), err)
				return false
			}
			c.SendCommand(g.GameStr, playtak.FormatServer(move))
			log.Printf("my-move game-id=%s ply=%d ptn=%d.%s move=%q",
				g.ID,
				g.p.MoveNumber(),
				g.p.MoveNumber()/2+1,
				strings.ToUpper(g.p.ToMove().String()[:1]),
				ptn.FormatMove(move))
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
		case "Tell":
			who, msg := playtak.ParseTell(line)
			if who != "" {
				g.bot.HandleTell(who, msg)
			}
		case "Shout":
			who, msg := playtak.ParseShout(line)
			if who != "" {
				g.bot.HandleChat("", who, msg)
			}
			continue
		case "ShoutRoom":
			room, who, msg := playtak.ParseShoutRoom(line)
			if who != "" {
				g.bot.HandleChat(room, who, msg)
			}
			continue
		default:
			continue
		}
		switch bits[1] {
		case "P", "M":
			move, err := playtak.ParseServer(strings.Join(bits[1:], " "))
			if err != nil {
				panic(err)
			}
			next, err := g.p.Move(move)
			if err != nil {
				panic(err)
			}
			log.Printf("their-move game-id=%s ply=%d ptn=%d.%s move=%q",
				g.ID,
				g.p.MoveNumber(),
				g.p.MoveNumber()/2+1,
				strings.ToUpper(g.p.ToMove().String()[:1]),
				ptn.FormatMove(move))
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
			if timeout != nil {
				return false
			}
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
