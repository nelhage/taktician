package tests

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"github.com/nelhage/taktician/ptn"
)

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
