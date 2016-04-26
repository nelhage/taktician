package ptn

import (
	"bufio"
	"errors"
	"io"
	"strconv"
	"strings"
	"unicode"

	"nelhage.com/tak/tak"
)

type Tag struct {
	Name  string
	Value string
}

type Op interface {
	op()

	Source() string
}

type opCommon struct {
	src string
}

func (o opCommon) Source() string {
	return o.src
}

func (o opCommon) op() {}

type MoveNumber struct {
	opCommon
	Number int
}

type Move struct {
	opCommon
	Move      tak.Move
	Modifiers string
}

type Comment struct {
	opCommon
	Comment string
}

type PTN struct {
	Tags []Tag
	Ops  []Op
}

func ParsePTN(r io.Reader) (*PTN, error) {
	buf := bufio.NewReader(r)
	var ptn PTN
	if err := readEvents(buf, &ptn); err != nil {
		return nil, err
	}
	if err := readMoves(buf, &ptn); err != nil && err != io.EOF {
		return nil, err
	}
	return &ptn, nil
}

func readEvents(r *bufio.Reader, ptn *PTN) error {
	for {
		if e := skipWS(r); e != nil {
			return e
		}
		c, e := r.ReadByte()
		if e != nil {
			return e
		}
		if c != '[' {
			return r.UnreadByte()
		}
		line, e := r.ReadString(']')
		if e != nil {
			return e
		}
		line = line[:len(line)-1]
		bits := strings.SplitN(line, " ", 2)
		if len(bits) != 2 {
			return errors.New("bad tag")
		}
		tag := Tag{
			Name:  bits[0],
			Value: strings.Trim(bits[1], "\""),
		}
		ptn.Tags = append(ptn.Tags, tag)
	}
}

func readMoves(r *bufio.Reader, ptn *PTN) error {
	s := bufio.NewScanner(r)
	s.Split(splitMoves)
	for s.Scan() {
		tok := s.Text()
		common := opCommon{tok}
		switch {
		case tok[0] == '{':
			ptn.Ops = append(ptn.Ops, &Comment{common, tok[1 : len(tok)-1]})
		case tok[len(tok)-1] == '.':
			n, e := strconv.Atoi(tok[:len(tok)-1])
			if e != nil {
				return e
			}
			ptn.Ops = append(ptn.Ops, &MoveNumber{common, n})
		default:
			trimmed := strings.TrimRight(tok, "?!'")
			move, e := ParseMove(trimmed)
			if e != nil {
				return e
			}
			ptn.Ops = append(ptn.Ops, &Move{common, *move, tok[len(trimmed):]})
		}
	}
	return s.Err()
}

func splitMoves(buf []byte, atEOF bool) (int, []byte, error) {
	start := 0
	for start < len(buf) && unicode.IsSpace(rune(buf[start])) {
		start++
	}
	if start == len(buf) {
		return start, nil, nil
	}
	if buf[start] == '{' {
		for i := start; i < len(buf); i++ {
			if buf[i] == '}' {
				return i + 1, buf[start : i+1], nil
			}
		}
	} else {
		for i := start; i < len(buf); i++ {
			if unicode.IsSpace(rune(buf[i])) {
				return i + 1, buf[start:i], nil
			}
		}
	}
	if atEOF {
		return len(buf), buf[start:], nil
	}
	return start, nil, nil
}

func skipWS(r *bufio.Reader) error {
	for {
		c, e := r.ReadByte()
		if e != nil {
			return e
		}
		if !unicode.IsSpace(rune(c)) {
			return r.UnreadByte()
		}
	}
}
