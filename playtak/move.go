package playtak

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/nelhage/taktician/tak"
)

func parseSquare(square string) (x, y int, err error) {
	if len(square) != 2 {
		return 0, 0, fmt.Errorf("bad coord `%s'", square)
	}
	if square[0] < 'A' || square[0] > 'H' {
		return 0, 0, fmt.Errorf("bad rank %s", square)
	}
	if square[1] < '1' || square[1] > '8' {
		return 0, 0, fmt.Errorf("bad file %s", square)
	}
	return int(square[0] - 'A'), int(square[1] - '1'), nil
}

func formatSquare(x, y int8) string {
	return string([]byte{byte(x) + 'A', byte(y) + '1'})
}

func ParseServer(server string) (tak.Move, error) {
	words := strings.Split(server, " ")
	switch words[0] {
	case "P":
		if len(words) != 2 && len(words) != 3 {
			return tak.Move{}, fmt.Errorf("command too short: %s", server)
		}
		x, y, err := parseSquare(words[1])
		if err != nil {
			return tak.Move{}, err
		}
		m := tak.Move{
			X: int8(x), Y: int8(y), Type: tak.PlaceFlat,
		}
		if len(words) == 3 {
			switch words[2] {
			case "C":
				m.Type = tak.PlaceCapstone
			case "W":
				m.Type = tak.PlaceStanding
			default:
				return tak.Move{}, fmt.Errorf("bad place: %s", server)
			}
		}
		return m, nil

	case "M":
		if len(words) < 4 {
			return tak.Move{}, fmt.Errorf("command too short: %s", server)
		}
		sx, sy, err := parseSquare(words[1])
		if err != nil {
			return tak.Move{}, err
		}
		ex, ey, err := parseSquare(words[2])
		if err != nil {
			return tak.Move{}, err
		}
		m := tak.Move{X: int8(sx), Y: int8(sy)}
		switch {
		case ex > sx && ey == sy:
			m.Type = tak.SlideRight
		case ex < sx && ey == sy:
			m.Type = tak.SlideLeft
		case ey > sy && ex == sx:
			m.Type = tak.SlideUp
		case ey < sy && ex == sx:
			m.Type = tak.SlideDown
		default:
			return tak.Move{}, fmt.Errorf("bad slide: %s", server)
		}
		m.Slides = make([]byte, len(words)-3)
		for i, drop := range words[3:] {
			n, e := strconv.Atoi(drop)
			if e != nil {
				return tak.Move{}, fmt.Errorf("bad drop: %s", server)
			}
			m.Slides[i] = byte(n)
		}
		return m, nil
	default:
		return tak.Move{}, fmt.Errorf("bad command: %s", server)
	}
}

func FormatServer(m *tak.Move) string {
	ex, ey := m.X, m.Y
	switch m.Type {
	case tak.PlaceFlat:
		return fmt.Sprintf("P %s", formatSquare(m.X, m.Y))
	case tak.PlaceCapstone:
		return fmt.Sprintf("P %s C", formatSquare(m.X, m.Y))
	case tak.PlaceStanding:
		return fmt.Sprintf("P %s W", formatSquare(m.X, m.Y))
	case tak.SlideRight:
		ex = m.X + int8(len(m.Slides))
	case tak.SlideLeft:
		ex = m.X - int8(len(m.Slides))
	case tak.SlideDown:
		ey = m.Y - int8(len(m.Slides))
	case tak.SlideUp:
		ey = m.Y + int8(len(m.Slides))
	}
	var out bytes.Buffer
	out.WriteString("M ")
	out.WriteString(formatSquare(m.X, m.Y))
	out.WriteString(" ")
	out.WriteString(formatSquare(ex, ey))
	for _, s := range m.Slides {
		fmt.Fprintf(&out, " %d", s)
	}
	return out.String()
}
