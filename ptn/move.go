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
	if len(move) < i+2 {
		return tak.Move{}, errors.New("move too short")
	}

	if move[i] >= 'a' && move[i] <= 'h' {
		m.X = int8(move[i] - 'a')
		i++
	} else {
		return tak.Move{}, errors.New("illegal move")
	}
	if move[i] >= '1' && move[i] <= '8' {
		m.Y = int8(move[i] - '1')
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
	j := 0
	var slides []int
	for ; i != len(move); i++ {
		d := move[i]
		slides = append(slides, int(d-'0'))
		j++
		stack -= int(d - '0')
	}
	if stack > 0 {
		slides = append(slides, stack)
	} else if stack < 0 {
		return tak.Move{}, errors.New("malformed move: bad count")
	}
	m.Slides = tak.MkSlides(slides...)

	return m, nil
}

func FormatMove(m *tak.Move) string {
	return formatMove(m, false)
}

func FormatMoveLong(m *tak.Move) string {
	return formatMove(m, true)
}

func formatMove(m *tak.Move, long bool) string {
	var out []byte
	stack := 0
	if !m.Slides.Empty() {
		for it, ok := m.Slides.Iterator(); ok; it, ok = it.Next() {
			stack += it.Elem()
		}
		if long || stack != 1 {
			out = append(out, byte('0'+stack))
		}
	}
	switch m.Type {
	case tak.PlaceFlat:
		if long {
			out = append(out, 'F')
		}
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
	if !m.Slides.Empty() && (long || m.Slides.Len() != 1) {
		for it, ok := m.Slides.Iterator(); ok; it, ok = it.Next() {
			out = append(out, byte('0'+it.Elem()))
		}
	}
	return string(out)
}
