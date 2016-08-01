package cli

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

type Player interface {
	GetMove(p *tak.Position) tak.Move
}

type GlyphSet struct {
	Flat     string
	Standing string
	Capstone string
}

type Glyphs struct {
	White, Black GlyphSet
}

type CLI struct {
	moves []tak.Move
	p     *tak.Position

	Config tak.Config
	Glyphs *Glyphs
	Out    io.Writer
	White  Player
	Black  Player
}

var DefaultGlyphs = Glyphs{
	White: GlyphSet{
		Flat:     "W",
		Standing: "WS",
		Capstone: "WC",
	},
	Black: GlyphSet{
		Flat:     "B",
		Standing: "BS",
		Capstone: "BC",
	},
}

var UnicodeGlyphs = Glyphs{
	White: GlyphSet{
		Flat:     "□",
		Standing: "║",
		Capstone: "♙",
	},
	Black: GlyphSet{
		Flat:     "▪",
		Standing: "┃",
		Capstone: "♟",
	},
}

func (c *CLI) Play() *tak.Position {
	c.moves = nil
	c.p = tak.New(c.Config)
	for {
		c.render()
		if ok, _ := c.p.GameOver(); ok {
			d := c.p.WinDetails()
			fmt.Fprintf(c.Out, "Game Over! ")
			if d.Winner == tak.NoColor {
				fmt.Fprintf(c.Out, "Draw.")
			} else {
				fmt.Fprintf(c.Out, "%s wins by ", d.Winner)
				switch d.Reason {
				case tak.RoadWin:
					fmt.Fprintf(c.Out, "building a road")
				case tak.FlatsWin:
					fmt.Fprintf(c.Out, "flats count")
				}
			}
			fmt.Fprintf(c.Out, "\nflats count: white=%d black=%d\n",
				d.WhiteFlats,
				d.BlackFlats)
			return c.p
		}
		var m tak.Move
		if c.p.ToMove() == tak.White {
			m = c.White.GetMove(c.p)
		} else {
			m = c.Black.GetMove(c.p)
		}
		p, e := c.p.Move(&m)
		if e != nil {
			fmt.Fprintln(c.Out, "illegal move:", e)
		} else {
			if c.p.ToMove() == tak.White {
				fmt.Fprintf(c.Out, "%d. %s", c.p.MoveNumber()/2+1, ptn.FormatMove(&m))
			} else {
				fmt.Fprintf(c.Out, "%d. ... %s", c.p.MoveNumber()/2+1, ptn.FormatMove(&m))
			}
			c.p = p
			c.moves = append(c.moves, m)
		}
	}
}

func (c *CLI) Moves() []tak.Move {
	return c.moves
}

func (c *CLI) render() {
	RenderBoard(c.Glyphs, c.Out, c.p)
}

func RenderBoard(g *Glyphs, out io.Writer, p *tak.Position) {
	if g == nil {
		g = &DefaultGlyphs
	}
	fmt.Fprintln(out)
	fmt.Fprintf(out, "[%s to play]\n", p.ToMove())
	w := tabwriter.NewWriter(out, 4, 8, 1, '\t', 0)
	for y := p.Size() - 1; y >= 0; y-- {
		fmt.Fprintf(w, "%c.\t", '1'+y)
		for x := 0; x < p.Size(); x++ {
			var stk []string
			for _, stone := range p.At(x, y) {
				switch stone {
				case tak.MakePiece(tak.White, tak.Flat):
					stk = append(stk, g.White.Flat)
				case tak.MakePiece(tak.White, tak.Standing):
					stk = append(stk, g.White.Standing)
				case tak.MakePiece(tak.White, tak.Capstone):
					stk = append(stk, g.White.Capstone)
				case tak.MakePiece(tak.Black, tak.Flat):
					stk = append(stk, g.Black.Flat)
				case tak.MakePiece(tak.Black, tak.Standing):
					stk = append(stk, g.Black.Standing)
				case tak.MakePiece(tak.Black, tak.Capstone):
					stk = append(stk, g.Black.Capstone)
				default:
					panic(fmt.Sprintf("bad stone %v", stone))
				}
			}
			fmt.Fprintf(w, "[%s]\t", strings.Join(stk, " "))
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
