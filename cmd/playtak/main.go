package main

import (
	"bufio"
	"flag"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"nelhage.com/tak/game"
)

var (
	white = flag.String("white", "human", "white player")
	black = flag.String("black", "human", "white player")
)

func parsePlayer(in *bufio.Reader, s string) Player {
	if s == "human" {
		return &cliPlayer{
			out: os.Stdout, in: in,
		}
	}
	if s == "rand" {
		return &randomAI{
			r: rand.New(rand.NewSource(0)),
		}
	}
	if strings.HasPrefix(s, "rand:") {
		i, err := strconv.Atoi(s[len("rand:"):])
		if err != nil {
			log.Fatal(err)
		}
		return &randomAI{
			r: rand.New(rand.NewSource(int64(i))),
		}
	}
	log.Fatalf("unparseable player: %s", s)
	return nil
}

func main() {
	flag.Parse()
	in := bufio.NewReader(os.Stdin)
	st := &state{
		p:   game.New(game.Config{Size: 5}),
		out: os.Stdout,
	}
	st.white = parsePlayer(in, *white)
	st.black = parsePlayer(in, *black)
	playTak(st)
}
