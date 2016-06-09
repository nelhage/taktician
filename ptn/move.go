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
	groups := moveRE.FindStringSubmatchIndex(move)
	if groups == nil {
		return tak.Move{}, errors.New("illegal move")
	}
	const (
		place = 2 * (iota + 1)
		carry
		position
		direction
		drops
	)
	x := move[groups[position]] - 'a'
	y := move[groups[position]+1] - '1'

	m := tak.Move{X: int(x), Y: int(y)}
	if groups[direction] == groups[direction+1] {
		// place a piece
		if groups[carry] != groups[carry+1] || groups[drops] != groups[drops+1] {
			return tak.Move{}, errors.New("can't carry or drop without a direction")
		}
		switch {
		case groups[place] == groups[place+1]:
			m.Type = tak.PlaceFlat
		case move[groups[place]] == 'F':
			m.Type = tak.PlaceFlat
		case move[groups[place]] == 'S':
			m.Type = tak.PlaceStanding
		case move[groups[place]] == 'C':
			m.Type = tak.PlaceCapstone
		default:
			panic("parser error")
		}
		return m, nil
	}

	// a slide
	stack := 1
	if groups[carry] != groups[carry+1] {
		stack = int(move[groups[carry]] - '0')
	}
	for i := groups[drops]; i != groups[drops+1]; i++ {
		d := move[i]
		m.Slides = append(m.Slides, byte(d-'0'))
		stack -= int(d - '0')
	}
	if stack > 0 {
		m.Slides = append(m.Slides, byte(stack))
	} else if stack < 0 {
		return tak.Move{}, errors.New("malformed move: bad count")
	}
	switch move[groups[direction]] {
	case '<':
		m.Type = tak.SlideLeft
	case '>':
		m.Type = tak.SlideRight
	case '+':
		m.Type = tak.SlideUp
	case '-':
		m.Type = tak.SlideDown
	default:
		panic("parser error")
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
