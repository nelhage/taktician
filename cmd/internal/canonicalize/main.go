package canonicalize

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/symmetry"
	"github.com/nelhage/taktician/tak"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s FILE.ptn\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	g, e := ptn.ParseFile(flag.Arg(0))
	if e != nil {
		log.Fatalf("read %s: %v", flag.Arg(0), e)
	}

	var ms []tak.Move
	for _, o := range g.Ops {
		if m, ok := o.(*ptn.Move); ok {
			ms = append(ms, m.Move)
		}
	}

	sz, e := strconv.ParseUint(g.FindTag("Size"), 10, 32)
	if e != nil {
		log.Fatalf("bad size: %v", e)
	}
	out, e := symmetry.Canonical(int(sz), ms)
	if e != nil {
		log.Fatalf("canonicalize: %v", e)
	}

	i := 0
	for _, o := range g.Ops {
		if m, ok := o.(*ptn.Move); ok {
			m.Move = out[i]
			i++
		}
	}

	fmt.Printf(g.Render())
}
