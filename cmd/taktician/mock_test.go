package main

import (
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nelhage/taktician/tak"
	"golang.org/x/net/context"
)

type Expectation struct {
	send, recv []string
}

type TestClient struct {
	send, recv chan string

	t      *testing.T
	expect []Expectation
}

func NewTestClient(t *testing.T, expect []Expectation) *TestClient {
	c := &TestClient{
		send:   make(chan string),
		recv:   make(chan string),
		t:      t,
		expect: expect,
	}
	go c.sendRecv()
	return c
}

func (t *TestClient) sendRecv() {
	for i, e := range t.expect {
		for _, s := range e.send {
			t.recv <- s
			log.Printf("[srv] -> %s", s)
		}
		for j, r := range e.recv {
			got := <-t.send
			log.Printf("[srv] <- %s", got)
			if got != r {
				t.t.Fatalf("msg %d,%d: got %q != %q",
					i, j, got, r)
			}
		}
	}
	close(t.recv)
}

func (t *TestClient) SendCommand(cmd ...string) {
	t.send <- strings.Join(cmd, " ")
}
func (t *TestClient) Recv() <-chan string {
	return t.recv
}

type BotBase struct {
	game *Game
}

func (t *BotBase) NewGame(g *Game) {
	t.game = g
}

func (t *BotBase) GameOver() {}

func (t *BotBase) AcceptUndo() bool {
	return false
}
func (t *BotBase) HandleChat(who, msg string) {}

type TestBotStatic struct {
	BotBase
	moves []tak.Move
}

func (t *TestBotStatic) GetMove(ctx context.Context,
	p *tak.Position,
	mine, theirs time.Duration) tak.Move {
	log.Printf("(*TestBot).GetMove(ply=%d color=%s)",
		p.MoveNumber(),
		p.ToMove(),
	)
	if p.ToMove() != t.game.color {
		return tak.Move{}
	}
	m := t.moves[0]
	t.moves = t.moves[1:]
	return m
}

type TestBotUndo struct {
	TestBotStatic
	undoPly int
}

func (t *TestBotUndo) GetMove(ctx context.Context,
	p *tak.Position,
	mine, theirs time.Duration) tak.Move {
	if p.MoveNumber() == t.undoPly+1 {
		<-ctx.Done()
		t.undoPly = -1
		return tak.Move{}
	}
	return t.TestBotStatic.GetMove(ctx, p, mine, theirs)
}

func (t *TestBotUndo) AcceptUndo() bool {
	return true
}

type TestBotThinker struct {
	TestBotStatic
	wg sync.WaitGroup
}

func (t *TestBotThinker) GetMove(ctx context.Context,
	p *tak.Position,
	mine, theirs time.Duration) tak.Move {
	t.wg.Add(1)
	defer t.wg.Done()
	if p.ToMove() != t.game.color {
		<-ctx.Done()
		return tak.Move{}
	}
	return t.TestBotStatic.GetMove(ctx, p, mine, theirs)
}
