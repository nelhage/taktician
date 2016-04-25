package cli

import (
	"fmt"
	"io"
	"text/tabwriter"

	"nelhage.com/tak/ptn"
	"nelhage.com/tak/tak"
)

type Player interface {
	GetMove(p *tak.Position) *tak.Move
}

type CLI struct {
	moves []tak.Move
	p     *tak.Position

	Config tak.Config
	Out    io.Writer
	White  Player
	Black  Player
}

func (c *CLI) Play() *tak.Position {
	c.p = tak.New(c.Config)
	for {
		c.render()
		if len(c.moves) > 0 && len(c.moves)%2 == 0 {
			fmt.Fprintf(c.Out,
				"%d. %s  %s\n",
				len(c.moves)/2,
				ptn.FormatMove(&c.moves[len(c.moves)-2]),
				ptn.FormatMove(&c.moves[len(c.moves)-1]))
		}
		if ok, w := c.p.GameOver(); ok {
			fmt.Fprintln(c.Out, "Game over! Winner:", w)
			return c.p
		}
		var m *tak.Move
		if c.p.ToMove() == tak.White {
			m = c.White.GetMove(c.p)
		} else {
			m = c.Black.GetMove(c.p)
		}
		p, e := c.p.Move(*m)
		if e != nil {
			fmt.Fprintln(c.Out, "illegal move:", e)
		} else {
			c.p = p
			c.moves = append(c.moves, *m)
		}
	}
}

func (c *CLI) render() {
	RenderBoard(c.Out, c.p)
}

func RenderBoard(out io.Writer, p *tak.Position) {
	fmt.Fprintln(out)
	fmt.Fprintf(out, "[%s to play]\n", p.ToMove())
	w := tabwriter.NewWriter(out, 4, 8, 1, '\t', 0)
	for y := p.Size() - 1; y >= 0; y-- {
		fmt.Fprintf(w, "%c.\t", '1'+y)
		for x := 0; x < p.Size(); x++ {
			fmt.Fprintf(w, "%v\t", p.At(x, y))
		}
		fmt.Fprintf(w, "\n")
	}
	fmt.Fprintf(w, "\t")
	for x := 0; x < p.Size(); x++ {
		fmt.Fprintf(w, "%c.\t", 'a'+x)
	}
	fmt.Fprintf(w, "\n")
	w.Flush()
	fmt.Fprintf(out, "stones: W:%d B:%d\n", p.WhiteStones(), p.BlackStones())
}
