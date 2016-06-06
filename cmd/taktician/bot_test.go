package main

import (
	"fmt"
	"testing"

	"github.com/nelhage/taktician/playtak"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

func parseMoves(spec [][2]string) [][2]*tak.Move {
	var out [][2]*tak.Move
	for _, r := range spec {
		var o [2]*tak.Move
		for i, n := range r {
			if n == "" {
				continue
			}
			m, e := ptn.ParseMove(n)
			if e != nil {
				panic("bad ptn")
			}
			o[i] = &m
		}
		out = append(out, o)
	}
	return out
}

func TestBasicGame(t *testing.T) {
	moves := parseMoves([][2]string{
		{"a1", "e1"},
		{"e3", "b1"},
		{"e2", "b2"},
		{"Ce4", "a2"},
		{"e5", ""},
	})
	bot := &TestBot{}
	for _, r := range moves {
		bot.moves = append(bot.moves, *r[0])
	}

	startLine := "Game Start 100 5 Taktician vs HonestJoe white 600"
	var transcript []Expectation
	tm := 600
	for _, r := range moves {
		transcript = append(transcript, Expectation{
			recv: []string{
				fmt.Sprintf("Game#100 %s", playtak.FormatServer(r[0])),
			},
		})
		if r[1] == nil {
			continue
		}
		transcript = append(transcript, Expectation{
			send: []string{
				fmt.Sprintf("Game#100 %s", playtak.FormatServer(r[1])),
				fmt.Sprintf("Game#100 Time %d %d", tm, tm),
			},
		})
		tm -= 10
	}
	transcript = append(transcript, Expectation{
		send: []string{
			"Game#100 Over R-0",
		},
	})

	c := NewTestClient(t, transcript)
	playGame(c, bot, startLine)
}
