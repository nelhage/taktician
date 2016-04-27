package ptn

import (
	"errors"
	"regexp"

	"nelhage.com/tak/tak"
)

var moveRE = regexp.MustCompile(
	// [place] [carry] position [direction] [drops] [top]
	`([CFS]?)([1-8]?)([a-h][1-9])([<>+-]?)([1-8]*)([CFS]?)`,
)

func ParseMove(move string) (tak.Move, error) {
	groups := moveRE.FindStringSubmatch(move)
	if groups == nil {
		return tak.Move{}, errors.New("illegal move")
	}
	var (
		place     = groups[1]
		carry     = groups[2]
		position  = groups[3]
		direction = groups[4]
		drops     = groups[5]
	)
	x := position[0] - 'a'
	y := position[1] - '1'

	m := tak.Move{X: int(x), Y: int(y)}
	if direction == "" {
		// place a piece
		if carry != "" || drops != "" {
			return tak.Move{}, errors.New("can't carry or drop without a direction")
		}
		switch place {
		case "F", "":
			m.Type = tak.PlaceFlat
		case "S":
			m.Type = tak.PlaceStanding
		case "C":
			m.Type = tak.PlaceCapstone
		default:
			panic("parser error")
		}
		return m, nil
	}

	// a slide
	stack := 1
	if carry != "" {
		stack = int(carry[0] - '0')
	}
	for _, d := range drops {
		m.Slides = append(m.Slides, byte(d-'0'))
		stack -= int(d - '0')
	}
	if stack > 0 {
		m.Slides = append(m.Slides, byte(stack))
	}
	switch direction {
	case "<":
		m.Type = tak.SlideLeft
	case ">":
		m.Type = tak.SlideRight
	case "+":
		m.Type = tak.SlideUp
	case "-":
		m.Type = tak.SlideDown
	default:
		panic("parser error")
	}

	return m, nil
}

func FormatMove(m *tak.Move) string {
	var out []byte
	if len(m.Slides) > 0 {
		stack := 0
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
	for i, s := range m.Slides {
		if i < len(m.Slides)-1 {
			out = append(out, byte('0'+s))
		}
	}
	return string(out)
}
