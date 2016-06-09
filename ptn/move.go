package ptn

import (
	"errors"
	"regexp"

	"github.com/nelhage/taktician/tak"
)

var moveRE = regexp.MustCompile(
	// [place] [carry] position [direction] [drops] [top]
	`\A([CFS]?)([1-8]?)([a-h][1-9])([<>+-]?)([1-8]*)([CFS]?)\z`,
)

func ParseMove(move string) (tak.Move, error) {
	if len(move) < 2 {
		return tak.Move{}, errors.New("move too short")
	}

	var m tak.Move
	var stack int
	i := 0
	switch move[i] {
	case 'F':
		m.Type = tak.PlaceFlat
		i++
	case 'S':
		m.Type = tak.PlaceStanding
		i++
	case 'C':
		m.Type = tak.PlaceCapstone
		i++
	default:
		if move[i] >= '1' && move[i] <= '8' {
			stack = int(move[i] - '0')
			i++
		} else {
			// provisional, may be updated if we see a
			// slide
			m.Type = tak.PlaceFlat
		}
	}
	if move[i] >= 'a' && move[i] <= 'h' {
		m.X = int(move[i] - 'a')
		i++
	} else {
		return tak.Move{}, errors.New("illegal move")
	}
	if move[i] >= '1' && move[i] <= '8' {
		m.Y = int(move[i] - '1')
		i++
	} else {
		return tak.Move{}, errors.New("illegal move")
	}
	if i == len(move) {
		if stack != 0 {
			return tak.Move{}, errors.New("illegal move")
		}
		return m, nil
	}
	switch move[i] {
	case '<':
		m.Type = tak.SlideLeft
	case '>':
		m.Type = tak.SlideRight
	case '+':
		m.Type = tak.SlideUp
	case '-':
		m.Type = tak.SlideDown
	default:
		return tak.Move{}, errors.New("bad move")
	}
	if stack == 0 {
		stack = 1
	}
	i++
	for ; i != len(move); i++ {
		d := move[i]
		m.Slides = append(m.Slides, byte(d-'0'))
		stack -= int(d - '0')
	}
	if stack > 0 {
		m.Slides = append(m.Slides, byte(stack))
	} else if stack < 0 {
		return tak.Move{}, errors.New("malformed move: bad count")
	}

	return m, nil
}

func FormatMove(m *tak.Move) string {
	var out []byte
	stack := 0
	if len(m.Slides) > 0 {
		for _, c := range m.Slides {
			stack += int(c)
		}
		if stack != 1 {
			out = append(out, byte('0'+stack))
		}
	}
	switch m.Type {
	case tak.PlaceFlat:
	case tak.PlaceCapstone:
		out = append(out, 'C')
	case tak.PlaceStanding:
		out = append(out, 'S')
	}
	out = append(out, byte('a'+m.X))
	out = append(out, byte('1'+m.Y))
	switch m.Type {
	case tak.SlideLeft:
		out = append(out, '<')
	case tak.SlideRight:
		out = append(out, '>')
	case tak.SlideUp:
		out = append(out, '+')
	case tak.SlideDown:
		out = append(out, '-')
	}
	if len(m.Slides) > 0 && int(m.Slides[0]) != stack {
		for _, s := range m.Slides {
			out = append(out, byte('0'+s))
		}
	}
	return string(out)
}
