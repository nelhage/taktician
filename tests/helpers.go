package tests

import (
	"io/ioutil"
	"log"
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
		g, e := ptn.ParseFile(path.Join(d, de.Name()))
		if e != nil {
			log.Printf("parse(%s): %v", de.Name(), e)
			continue
		}
		out = append(out, g)
	}
	return out, nil
}
