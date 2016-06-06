package main

import (
	"log"
	"strings"
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
		}
		for j, r := range e.recv {
			got := <-t.send
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

type TestBot struct {
	game  *Game
	moves []tak.Move
}

func (t *TestBot) NewGame(g *Game) {
	t.game = g
}

func (t *TestBot) GameOver() {}

func (t *TestBot) GetMove(ctx context.Context,
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

func (t *TestBot) AcceptUndo() bool {
	return false
}
func (t *TestBot) HandleChat(who, msg string) {}
