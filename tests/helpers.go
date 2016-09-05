package tests

import (
	"io/ioutil"
	"log"
	"path"
	"strings"

	"github.com/nelhage/taktician/ptn"
)

func readPTNs(d string) (map[string]*ptn.PTN, error) {
	ents, e := ioutil.ReadDir(d)
	if e != nil {
		return nil, e
	}
	out := make(map[string]*ptn.PTN)
	for _, de := range ents {
		if !strings.HasSuffix(de.Name(), ".ptn") {
			continue
		}
		g, e := ptn.ParseFile(path.Join(d, de.Name()))
		if e != nil {
			log.Printf("parse(%s): %v", de.Name(), e)
			continue
		}
		out[de.Name()] = g
	}
	return out, nil
}
