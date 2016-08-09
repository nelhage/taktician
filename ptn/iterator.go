package ptn

import "github.com/nelhage/taktician/tak"

type Iterator struct {
	ptn *PTN
	i   int

	err  error
	over bool

	initial  bool
	position *tak.Position
	ptnMove  int
	move     tak.Move
}

func (p *PTN) Iterator() *Iterator {
	pos, err := p.InitialPosition()
	return &Iterator{
		ptn:      p,
		position: pos,
		err:      err,
		initial:  true,
	}
}

func (i *Iterator) Err() error {
	return i.err
}

func (i *Iterator) apply() bool {
	next, e := i.position.Move(&i.move)
	if e != nil {
		i.err = e
		return false
	}
	i.position = next
	i.move = tak.Move{}
	return true
}

func (i *Iterator) Next() bool {
	if i.err != nil || i.over {
		return false
	}

	if i.move.Type != 0 {
		if !i.apply() {
			return false
		}
		if ok, _ := i.position.GameOver(); ok {
			return true
		}
	}

	for i.i < len(i.ptn.Ops) {
		op := i.ptn.Ops[i.i]
		i.i++
		switch o := op.(type) {
		case *MoveNumber:
			i.ptnMove = o.Number
		case *Move:
			i.move = o.Move
			return true
		}
	}
	i.over = true
	if i.move.Type != 0 {
		return i.apply()
	}
	return true
}

func (i *Iterator) Position() *tak.Position {
	return i.position
}

func (i *Iterator) PTNMove() int {
	return i.ptnMove
}

func (i *Iterator) PeekMove() tak.Move {
	return i.move
}
