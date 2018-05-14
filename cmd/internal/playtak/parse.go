package playtak

import (
	"regexp"
	"strings"
)

var commandRE = regexp.MustCompile(`^([^ :]+):?\s*([^ :]+):?\s*(.*)$`)

func parseCommand(whoami string, msg string) (string, string) {
	gs := commandRE.FindStringSubmatch(msg)
	if gs == nil {
		return "", ""
	}
	if !strings.EqualFold(gs[1], whoami) &&
		!strings.EqualFold(gs[1]+"bot", whoami) {
		return "", ""
	}
	return gs[2], gs[3]

}
