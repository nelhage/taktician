package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"

	"github.com/nelhage/taktician/cli"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

var games = flag.String("games", "", "Directory of .ptn files to self-check on")

func readPTNs(d string) ([]*ptn.PTN, error) {
	ents, e := ioutil.ReadDir(d)
	if e != nil {
		return nil, e
	}
	var out []*ptn.PTN
	for _, de := range ents {
		if !strings.HasSuffix(de.Name(), ".ptn") {
			continue
		}
		f, e := os.Open(path.Join(d, de.Name()))
		if e != nil {
			log.Printf("open(%s): %v", de.Name(), e)
			continue
		}
		g, e := ptn.ParsePTN(f)
		if e != nil {
			log.Printf("parse(%s): %v", de.Name(), e)
			f.Close()
			continue
		}
		f.Close()
		out = append(out, g)
	}
	return out, nil
}

func TestPlayPTNs(t *testing.T) {
	if *games == "" {
		t.SkipNow()
	}
	ptns, err := readPTNs(*games)
	if err != nil {
		t.Fatalf("read ptns: %v", err)
	}
	for _, p := range ptns {
		playPTN(t, p)
	}
}

func playPTN(t *testing.T, p *ptn.PTN) {
	size, _ := strconv.Atoi(p.FindTag("Size"))
	g := tak.New(tak.Config{Size: size})
	for _, op := range p.Ops {
		if m, ok := op.(*ptn.Move); ok {
			next, e := g.Move(&m.Move)
			if e != nil {
				fmt.Printf("illegal move: %s\n", ptn.FormatMove(&m.Move))
				fmt.Printf("move=%d\n", g.MoveNumber())
				cli.RenderBoard(os.Stdout, g)
				t.Fatal("illegal move")
			}
			g = next
		}
	}
}
