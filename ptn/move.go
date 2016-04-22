package ptn

import (
	"errors"
	"regexp"

	"nelhage.com/tak/game"
)

var moveRE = regexp.MustCompile(
	// [place] [carry] position [direction] [drops] [top]
	`([CFS]?)([1-8]?)([a-h][1-9])([<>+-]?)([1-8]*)([CFS]?)`,
)

func ParseMove(move string) (*game.Move, error) {
	groups := moveRE.FindStringSubmatch(move)
	if groups == nil {
		return nil, errors.New("illegal move")
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

	m := &game.Move{X: int(x), Y: int(y)}
	if direction == "" {
		// place a piece
		if carry != "" || drops != "" {
			return nil, errors.New("can't carry or drop without a direction")
		}
		switch place {
		case "F", "":
			m.Type = game.PlaceFlat
		case "S":
			m.Type = game.PlaceStanding
		case "C":
			m.Type = game.PlaceCapstone
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
		m.Type = game.SlideLeft
	case ">":
		m.Type = game.SlideRight
	case "+":
		m.Type = game.SlideUp
	case "-":
		m.Type = game.SlideDown
	default:
		panic("parser error")
	}

	return m, nil
}
