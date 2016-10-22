package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

var (
	debug = flag.Int("debug", 0, "debug level")
)

func main() {
	flag.Parse()

	scan := bufio.NewScanner(os.Stdin)
	if !scan.Scan() {
		log.Fatal("reading game", scan.Err())
	}
	bits := strings.Split(strings.TrimRight(scan.Text(), "\n\r"), " ")
	no, _ := strconv.Atoi(bits[0])
	size, _ := strconv.Atoi(bits[1])
	tm, _ := strconv.Atoi(bits[2])

	var color tak.Color
	switch no {
	case 1:
		color = tak.White
	case 2:
		color = tak.Black
	default:
		log.Fatal("bad number:", bits[0])
	}

	playGame(scan, color, size, time.Duration(tm)*time.Second)
}

func playGame(scan *bufio.Scanner, color tak.Color, size int, timeLimit time.Duration) {
	ctx := context.Background()
	p := tak.New(tak.Config{
		Size: size,
	})
	taktician := ai.NewMinimax(ai.MinimaxConfig{
		Size:       size,
		NoMultiCut: true,
		Debug:      *debug,
	})
	remaining := timeLimit
	for {
		if ok, _ := p.GameOver(); ok {
			break
		}
		var e error
		var move tak.Move
		if p.ToMove() == color {
			start := time.Now()
			budget := timeLimit / 30
			if budget > remaining/10 {
				budget = remaining / 10
			}

			ctx, cancel := context.WithTimeout(ctx, budget)
			move = taktician.GetMove(ctx, p)
			cancel()
			fmt.Println(ptn.FormatMoveLong(&move))

			remaining = remaining - time.Now().Sub(start)
		} else {
			if !scan.Scan() {
				log.Fatal("read move:", scan.Err())
			}
			move, e = ptn.ParseMove(strings.TrimRight(scan.Text(), "\n\r"))
			if e != nil {
				log.Fatalf("parse move %s: %v",
					scan.Text(), e)
			}
		}
		p, e = p.Move(&move)
		if e != nil {
			log.Fatalf("move %s: %v",
				ptn.FormatMove(&move),
				e)
		}
	}
}
