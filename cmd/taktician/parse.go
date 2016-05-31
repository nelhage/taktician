package main

import (
	"regexp"
	"strings"
)

var commandRE = regexp.MustCompile(`^([^ :]+):?\s*([^ :]+):?\s*(.*)$`)

func parseCommand(msg string) (string, string) {
	gs := commandRE.FindStringSubmatch(msg)
	if gs == nil {
		return "", ""
	}
	if !strings.EqualFold(gs[1], *user) &&
		!strings.EqualFold(gs[1]+"bot", *user) {
		return "", ""
	}
	return gs[2], gs[3]

}
