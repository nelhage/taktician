package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/nelhage/taktician/canonicalize"
	"github.com/nelhage/taktician/logs"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

var (
	logDir = flag.String("games", "games", "game log directory")
)

var (
	size      = flag.Int("size", 5, "what size to analyze")
	minRating = flag.Int("rating", 1600, "minimum rating to consider")
	minCount  = flag.Int("count", 100, "render games with >= [this many] moves")
	maxDepth  = flag.Int("depth", 8, "track tree to this many plies")
)

func main() {
	flag.Parse()

	repo, e := logs.Open(flag.Arg(0))
	if e != nil {
		log.Fatalf("parse %s: %v", flag.Arg(0), e)
	}
	defer repo.Close()
	sql := repo.DB()

	rows, e := sql.Query(
		`
SELECT day, id
FROM games g, ratings r1, ratings r2
WHERE r1.name = g.player1
 AND r2.name = g.player2
 AND NOT r1.bot AND NOT r2.bot
 AND r1.rating >= ?
 AND r2.rating >= ?
 AND g.size = ?
`, *minRating, *minRating, *size)
	if e != nil {
		log.Fatal("select: ", e)
	}
	defer rows.Close()

	tree := &tree{}

	for rows.Next() {
		var day string
		var id int
		e = rows.Scan(&day, &id)
		if e != nil {
			panic(e)
		}

		ptnPath := path.Join(*logDir, day, fmt.Sprintf("%d.ptn", id))
		g, e := ptn.ParseFile(ptnPath)
		if e != nil {
			log.Printf("parse %s: %v", ptnPath, e)
			continue
		}
		p, e := g.PositionAtMove(0, tak.NoColor)
		if e != nil {
			log.Printf("parse %s: %v", ptnPath, e)
			continue
		}

		var ms []tak.Move
		for _, o := range g.Ops {
			if m, ok := o.(*ptn.Move); ok {
				ms = append(ms, m.Move)
				if len(ms) >= 10 {
					break
				}
			}
		}
		ms, e = canonicalize.Canonical(p.Size(), ms)
		if e != nil {
			log.Printf("%s: %v", ptnPath, e)
			continue
		}

		if len(ms) > *maxDepth {
			ms = ms[:*maxDepth]
		}

		result := ptn.Result{Result: g.FindTag("Result")}

		insertTree(tree, ms, result.Winner())
	}

	bs, _ := json.Marshal(tree)
	ioutil.WriteFile("gametree.json", bs, 0644)
	f, e := os.Create("gametree.dot")
	defer f.Close()
	writeTree(f, tree)
	fmt.Printf("Common openings:\n")
	printLines(tree)

	line := minmax(tree)
	fmt.Printf("Best line:\n")
	for i, t := range line {
		dots := ""
		if i%2 == 1 {
			dots = ".. "
		}
		fmt.Printf("%d. %s%s %d-%d %0.0f%%\n",
			i/2+1, dots, t.Move, t.White, t.Black,
			100*float64(t.White)/float64(t.Count))
	}
}

type tree struct {
	id int

	Move     string  `json:",omitempty"`
	Children []*tree `json:",omitempty"`
	Count    int
	White    int
	Black    int
}

var nextID = 1

func insertTree(t *tree, ms []tak.Move, winner tak.Color) {
	t.Count++
	switch winner {
	case tak.White:
		t.White++
	case tak.Black:
		t.Black++
	}
	if len(ms) == 0 {
		return
	}
	var child *tree
	m := ptn.FormatMove(&ms[0])
	for _, ch := range t.Children {
		if ch.Move == m {
			child = ch
			break
		}
	}
	if child == nil {
		child = &tree{Move: m, id: nextID}
		nextID++
		t.Children = append(t.Children, child)
	}
	insertTree(child, ms[1:], winner)
}

func writeTree(f io.Writer, t *tree) {
	fmt.Fprintf(f, "digraph G {\n")
	writeTreeNode(0, f, t)
	fmt.Fprintf(f, "}\n")
}

func writeTreeNode(ply int, f io.Writer, t *tree) {
	var mno string
	move := ply/2 + 1
	if ply%2 == 0 {
		mno = fmt.Sprintf("%d. ", move)
	} else {
		mno = fmt.Sprintf("%d. .. ", move)
	}

	fmt.Fprintf(f, `  n%d [shape=box, label="%s %d-%d/%0.0f%%"]`,
		t.id, t.Move, t.White, t.Black, 100*float64(t.White)/float64(t.Count))
	fmt.Fprintln(f)
	for _, ch := range t.Children {
		if ch.Count < *minCount {
			continue
		}
		fmt.Fprintf(f, `  n%d -> n%d [label="%s%s %d/%0.0f%%"]`,
			t.id, ch.id, mno, ch.Move,
			ch.Count, 100*float64(ch.Count)/float64(t.Count))
		fmt.Fprintln(f)
		writeTreeNode(ply+1, f, ch)
	}
}

func printLines(t *tree) {
	walkLines([]*tree{}, t)
}

func walkLines(line []*tree, t *tree) {
	found := false
	for _, ch := range t.Children {
		if ch.Count >= *minCount && float64(ch.Count) >= 0.05*float64(t.Count) {
			walkLines(append(line, t), ch)
			found = true
		}
	}
	if !found {
		for _, m := range line {
			if m.Move == "" {
				continue
			}
			fmt.Printf("%s ", m.Move)
		}
		fmt.Printf("%s\n", t.Move)
	}
}

func minmax(t *tree) []*tree {
	var line []*tree
	who := tak.White
	for t != nil {
		var best *tree
		var max float64 = -1
		for _, ch := range t.Children {
			if ch.Count < *minCount {
				continue
			}
			var wins int
			if who == tak.White {
				wins = ch.White
			} else {
				wins = ch.Black
			}
			score := float64(wins) / float64(ch.Count)
			if score > max {
				max = score
				best = ch
			}
		}
		if best != nil {
			line = append(line, best)
		}
		t = best
		who = who.Flip()
	}
	return line
}
