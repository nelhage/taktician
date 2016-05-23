package tak

import "fmt"

type position3 struct {
	Position
	alloc struct {
		Height [3 * 3]uint8
		Stacks [3 * 3]uint64
		Groups [6]uint64
	}
}

type position4 struct {
	Position
	alloc struct {
		Height [4 * 4]uint8
		Stacks [4 * 4]uint64
		Groups [8]uint64
	}
}

type position5 struct {
	Position
	alloc struct {
		Height [5 * 5]uint8
		Stacks [5 * 5]uint64
		Groups [10]uint64
	}
}

type position6 struct {
	Position
	alloc struct {
		Height [6 * 6]uint8
		Stacks [6 * 6]uint64
		Groups [12]uint64
	}
}

type position7 struct {
	Position
	alloc struct {
		Height [7 * 7]uint8
		Stacks [7 * 7]uint64
		Groups [14]uint64
	}
}

type position8 struct {
	Position
	alloc struct {
		Height [8 * 8]uint8
		Stacks [8 * 8]uint64
		Groups [16]uint64
	}
}

func alloc(tpl *Position) *Position {
	switch tpl.Size() {
	case 3:
		a := &position3{Position: *tpl}
		a.Height = a.alloc.Height[:]
		a.Stacks = a.alloc.Stacks[:]
		a.analysis.WhiteGroups = a.alloc.Groups[:0]
		copy(a.Height, tpl.Height)
		copy(a.Stacks, tpl.Stacks)

		return &a.Position
	case 4:
		a := &position4{Position: *tpl}
		a.Height = a.alloc.Height[:]
		a.Stacks = a.alloc.Stacks[:]
		a.analysis.WhiteGroups = a.alloc.Groups[:0]
		copy(a.Height, tpl.Height)
		copy(a.Stacks, tpl.Stacks)

		return &a.Position
	case 5:
		a := &position5{Position: *tpl}
		a.Height = a.alloc.Height[:]
		a.Stacks = a.alloc.Stacks[:]
		a.analysis.WhiteGroups = a.alloc.Groups[:0]
		copy(a.Height, tpl.Height)
		copy(a.Stacks, tpl.Stacks)

		return &a.Position
	case 6:
		a := &position6{Position: *tpl}
		a.Height = a.alloc.Height[:]
		a.Stacks = a.alloc.Stacks[:]
		a.analysis.WhiteGroups = a.alloc.Groups[:0]
		copy(a.Height, tpl.Height)
		copy(a.Stacks, tpl.Stacks)

		return &a.Position
	case 7:
		a := &position7{Position: *tpl}
		a.Height = a.alloc.Height[:]
		a.Stacks = a.alloc.Stacks[:]
		a.analysis.WhiteGroups = a.alloc.Groups[:0]
		copy(a.Height, tpl.Height)
		copy(a.Stacks, tpl.Stacks)

		return &a.Position
	case 8:
		a := &position8{Position: *tpl}
		a.Height = a.alloc.Height[:]
		a.Stacks = a.alloc.Stacks[:]
		a.analysis.WhiteGroups = a.alloc.Groups[:0]
		copy(a.Height, tpl.Height)
		copy(a.Stacks, tpl.Stacks)

		return &a.Position
	default:
		panic(fmt.Sprintf("illegal size: %d", tpl.Size()))
	}
}

func copyPosition(p *Position, out *Position) {
	h := out.Height
	s := out.Stacks
	g := out.analysis.WhiteGroups

	*out = *p
	out.Height = h
	out.Stacks = s
	out.analysis.WhiteGroups = g[:0]

	copy(out.Height, p.Height)
	copy(out.Stacks, p.Stacks)
}

func Alloc(size int) *Position {
	p := Position{cfg: &Config{Size: size}}
	return alloc(&p)
}
