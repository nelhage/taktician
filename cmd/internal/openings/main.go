package openings

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/google/subcommands"
	"github.com/nelhage/taktician/logs"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/symmetry"
	"github.com/nelhage/taktician/tak"
)

type Command struct {
	size      int
	minRating int
	minCount  int
	maxDepth  int
}

func (*Command) Name() string     { return "openings" }
func (*Command) Synopsis() string { return "Analyze openings from the playtak DB" }
func (*Command) Usage() string {
	return `openings [flags] GAMES.db
`
}

func (c *Command) SetFlags(flags *flag.FlagSet) {
	flags.IntVar(&c.size, "size", 5, "what size to analyze")
	flags.IntVar(&c.minRating, "rating", 1600, "minimum rating to consider")
	flags.IntVar(&c.minCount, "count", 100, "render games with >= [this many] moves")
	flags.IntVar(&c.maxDepth, "depth", 8, "track tree to this many plies")
}

func (c *Command) Execute(ctx context.Context, flag *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	repo, e := logs.Open(flag.Arg(0))
	if e != nil {
		log.Fatalf("open %s: %v", flag.Arg(0), e)
	}
	defer repo.Close()
	sql := repo.DB()

	rows, e := sql.Query(
		`
SELECT g.id, p.ptn
FROM games g, ratings r1, ratings r2, ptns p
WHERE r1.name = g.player_white
 AND r2.name = g.player_black
 AND NOT r1.bot AND NOT r2.bot
 AND r1.rating >= ?
 AND r2.rating >= ?
 AND g.size = ?
 AND p.id = g.id
 AND p.id IS NOT NULL
`, c.minRating, c.minRating, c.size)
	if e != nil {
		log.Fatal("select: ", e)
	}
	defer rows.Close()

	tree := &tree{}

	for rows.Next() {
		var id int
		var notation string
		e = rows.Scan(&id, &notation)
		if e != nil {
			panic(e)
		}

		g, e := ptn.ParsePTN(strings.NewReader(notation))
		if e != nil {
			log.Printf("parse %d: %v", id, e)
			continue
		}

		p, e := g.PositionAtMove(0, tak.NoColor)
		if e != nil {
			log.Printf("parse %d: %v", id, e)
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
		ms, e = symmetry.Canonical(p.Size(), ms)
		if e != nil {
			log.Printf("%d: %v", id, e)
			continue
		}

		if len(ms) > c.maxDepth {
			ms = ms[:c.maxDepth]
		}

		result := ptn.Result{Result: g.FindTag("Result")}

		insertTree(tree, ms, result.Winner())
	}

	bs, _ := json.Marshal(tree)
	ioutil.WriteFile("gametree.json", bs, 0644)
	f, e := os.Create("gametree.dot")
	defer f.Close()
	c.writeTree(f, tree)
	fmt.Printf("Common openings:\n")
	c.printLines(tree)

	line := c.minmax(tree)
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

	return subcommands.ExitSuccess
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
	m := ptn.FormatMove(ms[0])
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

func (c *Command) writeTree(f io.Writer, t *tree) {
	fmt.Fprintf(f, "digraph G {\n")
	c.writeTreeNode(0, f, t)
	fmt.Fprintf(f, "}\n")
}

func (c *Command) writeTreeNode(ply int, f io.Writer, t *tree) {
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
		if ch.Count < c.minCount {
			continue
		}
		fmt.Fprintf(f, `  n%d -> n%d [label="%s%s %d/%0.0f%%"]`,
			t.id, ch.id, mno, ch.Move,
			ch.Count, 100*float64(ch.Count)/float64(t.Count))
		fmt.Fprintln(f)
		c.writeTreeNode(ply+1, f, ch)
	}
}

func (c *Command) printLines(t *tree) {
	c.walkLines([]*tree{}, t)
}

func (c *Command) walkLines(line []*tree, t *tree) {
	found := false
	for _, ch := range t.Children {
		if ch.Count >= c.minCount && float64(ch.Count) >= 0.05*float64(t.Count) {
			c.walkLines(append(line, t), ch)
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

func (c *Command) minmax(t *tree) []*tree {
	var line []*tree
	who := tak.White
	for t != nil {
		var best *tree
		var max float64 = -1
		for _, ch := range t.Children {
			if ch.Count < c.minCount {
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
