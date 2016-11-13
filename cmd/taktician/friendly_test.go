package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nelhage/taktician/playtak"
	"github.com/nelhage/taktician/playtak/bot"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
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
		fpa:    true,
	}
	friendly.NewGame(&bot.Game{
		ID:       "123",
		GameStr:  "Game#123",
		Opponent: "nelhage",
		Color:    tak.Black,
		Size:     5,
	})

	mock.cmds = nil

	p := tak.New(tak.Config{Size: 5})
	m, _ := ptn.ParseMove("a1")
	p, _ = p.Move(&m)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	friendly.GetMove(ctx, p, time.Minute, time.Minute)
	cancel()

	if len(mock.cmds) != 2 {
		t.Fatalf("got commands: %#v", mock.cmds)
		return
	}
	if mock.cmds[0] != "Game#123 Resign" {
		t.Fatalf("Expected resign, got %q", mock.cmds[0])
	}

	mock.cmds = nil

	p = tak.New(tak.Config{Size: 5})
	m, _ = ptn.ParseMove("c3")
	p, _ = p.Move(&m)

	ctx, cancel = context.WithTimeout(context.Background(), 50*time.Millisecond)
	friendly.GetMove(ctx, p, time.Minute, time.Minute)
	cancel()
	if len(mock.cmds) != 0 {
		t.Errorf("sent commands: %#v", mock.cmds)
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
	}

	for _, tc := range cases {
		tc := tc
		t.Run(fmt.Sprintf("%d:%s", tc.size, tc.move), func(t *testing.T) {
			friendly := &Friendly{
				fpa: true,
				g: &bot.Game{
					ID:       "123",
					GameStr:  "Game#123",
					Opponent: "nelhage",
				},
			}

			p := tak.New(tak.Config{Size: tc.size})
			m, e := ptn.ParseMove(tc.move)
			if e != nil {
				t.Fatal("bad move", tc.move)
			}
			p, e = p.Move(&m)
			if e != nil {
				t.Fatal("bad move", tc.move)
			}

			ok := friendly.fpaWhiteOK(p)
			if ok != tc.ok {
				t.Fatalf("got %v want %v", ok, tc.ok)
			}
		})
	}
}
