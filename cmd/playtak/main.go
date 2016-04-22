package main

import (
	"bufio"
	"os"

	"nelhage.com/tak/game"
)

func main() {
	st := &state{
		p:   game.New(game.Config{Size: 5}),
		out: os.Stdout,
		in:  bufio.NewReader(os.Stdin),
	}
	playTak(st)
}
