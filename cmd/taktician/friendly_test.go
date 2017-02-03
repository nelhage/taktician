package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nelhage/taktician/playtak"
	"github.com/nelhage/taktician/playtak/bot"
	"github.com/nelhage/taktician/tak"
	"github.com/nelhage/taktician/taktest"
)

type mockClient struct {
	cmds []string
}

func (m *mockClient) SendCommand(args ...string) {
	m.cmds = append(m.cmds, strings.Join(args, " "))
}

func (m *mockClient) Recv() <-chan string {
	return nil
}

func (m *mockClient) Error() error {
	return nil
}

func (m *mockClient) Shutdown() {
}

func TestFPAResign(t *testing.T) {
	mock := &mockClient{}
	friendly := &Friendly{
		client: &playtak.Commands{mock},
		fpa:    &CenterBlack{},
	}
	game := &bot.Game{
		ID:       "123",
		GameStr:  "Game#123",
		Opponent: "nelhage",
		Color:    tak.Black,
		Size:     5,
	}
	game.Positions = []*tak.Position{
		taktest.Position(5, ""),
		taktest.Position(5, "a1"),
	}
	game.Moves = []tak.Move{
		taktest.Move("a1"),
	}
	friendly.NewGame(game)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	friendly.GetMove(ctx, game.Positions[1], time.Minute, time.Minute)
	cancel()

	if len(mock.cmds) != 4 {
		t.Fatalf("got commands: %#v", mock.cmds)
		return
	}
	if mock.cmds[2] != "Game#123 Resign" {
		t.Fatalf("Expected resign, got %q", mock.cmds[0])
	}

	mock.cmds = nil

	game.Positions = []*tak.Position{
		taktest.Position(5, ""),
		taktest.Position(5, "c3"),
	}
	game.Moves = []tak.Move{
		taktest.Move("c3"),
	}
	friendly.NewGame(game)

	ctx, cancel = context.WithTimeout(context.Background(), 50*time.Millisecond)
	friendly.GetMove(ctx, game.Positions[1], time.Minute, time.Minute)
	cancel()
	if len(mock.cmds) != 2 {
		t.Errorf("sent commands: %#v", mock.cmds)
	}
	for _, cmd := range mock.cmds {
		if !strings.HasPrefix(cmd, "Tell nelhage") {
			t.Errorf("sent commands: %#v", mock.cmds)
		}
	}
}

func TestFPAOK(t *testing.T) {
	cases := []struct {
		size int
		move string
		ok   bool
	}{
		{5, "c3", true},
		{5, "c2", false},
		{5, "a1", false},

		{6, "a1", false},
		{6, "b2", false},

		{6, "c3", true},
		{6, "c4", true},
		{6, "d3", true},
		{6, "d4", true},

		{6, "e3", false},
		{6, "b3", false},
		{6, "c2", false},
		{6, "c5", false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(fmt.Sprintf("%d:%s", tc.size, tc.move), func(t *testing.T) {
			rule := &CenterBlack{}
			p := tak.New(tak.Config{Size: tc.size})
			m := taktest.Move(tc.move)

			err := rule.LegalMove(p, m)
			if (err == nil) != tc.ok {
				t.Fatalf("got %v want %v", err, tc.ok)
			}
		})
	}
}
