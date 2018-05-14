package playtak

import (
	"fmt"
	"strings"

	"github.com/nelhage/taktician/ai"
)

var books []*ai.OpeningBook

const book5 = `
a1 e5 e4
a1 e5 d4
a1 e5 e3
a1 e1 c3
a1 e1 e2 d4
a1 e1 e2 e3 d2
a1 e1 e2 a2 e3
a1 e1 e2 Ce3 d2 d3
a1 e1 e2 a3 e3
a1 e1 e3 e2
a1 e1 d2 a2
a1 e1 e4
`

const book6 = `
a1 f6 e4
a1 f6 d4 d3 c4
a1 f1 e3 d4
a1 f1 e3 e2
a1 f1 e3 d3
a1 f1 e3 e4
a1 f1 e3 Cd4
a1 f1 f2
a1 f1 d4
a1 f1 d3 c3 d4
a1 f1 d3 d4
`

func init() {
	books = make([]*ai.OpeningBook, 9)
	var e error
	books[5], e = ai.BuildOpeningBook(5,
		strings.Split(strings.Trim(book5, " \n"), "\n"))
	if e == nil {
		books[6], e = ai.BuildOpeningBook(6,
			strings.Split(strings.Trim(book6, " \n"), "\n"))
	}
	if e != nil {
		panic(fmt.Sprintf("build: %v", e))
	}
}

func (c *Command) wrapWithBook(size int, p ai.TakPlayer) ai.TakPlayer {
	if !c.book {
		return p
	}
	if size != 5 && size != 6 {
		return p
	}
	return ai.WithOpeningBook(p, books[size])
}
