package playtak

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
		cmd:    &Command{},
		client: &playtak.Commands{"", mock},
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

func TestDoubleStackSelfCheck(t *testing.T) {
	sizes := []int{4, 5, 6}
	for _, s := range sizes {
		t.Run(fmt.Sprintf("size-%d", s), func(t *testing.T) {
			ds := &DoubleStack{}
			p := tak.New(tak.Config{Size: s})

		loop:
			for {
				m, ok := ds.GetMove(p)
				if !ok {
					switch p.MoveNumber() {
					case 0:
						m = tak.Move{
							X: 0, Y: 0,
							Type: tak.PlaceFlat,
						}
					case 1:
						m = tak.Move{
							X: int8(s - 1), Y: int8(s - 1),
							Type: tak.PlaceFlat,
						}
					default:
						break loop
					}
				}
				err := ds.LegalMove(p, m)
				if err != nil {
					t.Fatalf("[%d] returned bad move: %v",
						p.MoveNumber(), err)
				}
				t.Logf("[%d] m=%s", p.MoveNumber(), ptn.FormatMove(m))
				np, err := p.Move(m)
				if err != nil {
					t.Fatalf("[%d] returned illegal move: %v",
						p.MoveNumber(), err)
				}
				p = np
			}
			if p.MoveNumber() != 6 {
				t.Fatalf("did not return 6 moves")
			}
		})
	}
}

func TestDoubleStackLegal(t *testing.T) {
	cases := []struct {
		size  int
		moves string
		ok    bool
	}{
		{5, "a1 e1 e1+ a2 e2- a2-", true},
		{5, "a1 e5 e5- b1 e4+ b1<", true},
		{5, "a1 e5 e4", false},
		{5, "a1 e5 c3", false},
		{5, "a1 e5 e5- b2", false},
		{5, "a1 e5 e5< c3", false},
		{5, "a1 e1 e1+ a2 e2<", false},
		{5, "a1 e1 e1+ a2 e3", false},
		{5, "a1 e5 e5- b1 e4+ a1+", false},
		{5, "a1 e5 e5- b1 e4+ b1+", false},
		{5, "a1 e5 e5- b1 e4+ c1", false},

		{6, "a1 f1 f1+ a2 f2- a2-", true},
	}
	for _, tc := range cases {
		t.Run(tc.moves, func(t *testing.T) {
			ds := &DoubleStack{}
			moves := taktest.Moves(tc.moves)
			p := tak.New(tak.Config{Size: tc.size})
			for i, m := range moves {
				err := ds.LegalMove(p, m)

				if err != nil {
					if i != len(moves)-1 {
						t.Fatalf("early fail %s: %v",
							ptn.FormatMove(m),
							err,
						)
					}
					if tc.ok {
						t.Fatalf("false fail %s: %v",
							ptn.FormatMove(m),
							err,
						)
					}
				}

				p, err = p.Move(m)
				if err != nil {
					panic(err)
				}
			}
		})
	}
}

func TestAdjacent(t *testing.T) {
	cases := []struct {
		setup    string
		position string
	}{
		{"a1", "a1"},
		{"a1", "a5"},
		{"a1", "e5"},
		{"a1", "e1"},

		{"a1 a2", "a1"},
		{"a1 b1", "a1"},

		{"e1 e2", "e1"},
		{"e1 d1", "e1"},

		{"a5 a4", "a5"},
		{"a5 b5", "a5"},

		{"e5 e4", "e5"},
		{"e5 d5", "e5"},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s/%s", tc.setup, tc.position), func(t *testing.T) {
			p := taktest.Position(5, tc.setup)
			m := taktest.Move(tc.position)
			ax, ay := adjacent(p, int(m.X), int(m.Y))
			dx := ax - int(m.X)
			dy := ay - int(m.Y)
			if dx < 0 {
				dx = -dx
			}
			if dy < 0 {
				dy = -dy
			}

			if ax < 0 || ay < 0 || ax >= 5 || ay >= 5 {
				t.Fatalf("out of bounds (%d, %d)", ax, ay)
			}

			if !((dx == 1 && dy == 0) ||
				(dx == 0 && dy == 1)) {
				t.Fatalf("not adjacent (%d, %d)", ax, ay)
			}

			if p.Top(ax, ay) != 0 {
				t.Fatalf("occupied (%d, %d)", ax, ay)
			}
		})
	}
}
