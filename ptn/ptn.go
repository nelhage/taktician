package ptn

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
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

type GameOver struct {
	opCommon
	End tak.WinDetails
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

func (p *PTN) FindTag(name string) string {
	for _, t := range p.Tags {
		if t.Name == name {
			return t.Value
		}
	}
	return ""
}

func (p *PTN) InitialPosition() (*tak.Position, error) {
	sizeTag := p.FindTag("Size")
	size, e := strconv.Atoi(sizeTag)
	if e != nil {
		return nil, fmt.Errorf("bad size: %s", sizeTag)
	}
	tps := p.FindTag("TPS")
	var out *tak.Position
	if tps == "" {
		out = tak.New(tak.Config{Size: size})
	} else {
		out, e = ParseTPS(tps)
		if e != nil {
			return nil, fmt.Errorf("bad TPS: %v", e)
		}
		if out.Size() != size {
			return nil, fmt.Errorf("size mismatch: tag %d != TPS %d",
				size, out.Size())
		}
	}
	return out, nil
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
		case tok == "R-0":
			ptn.Ops = append(ptn.Ops, &GameOver{common,
				tak.WinDetails{Winner: tak.White, Reason: tak.RoadWin}})
		case tok == "0-R":
			ptn.Ops = append(ptn.Ops, &GameOver{common,
				tak.WinDetails{Winner: tak.Black, Reason: tak.RoadWin}})
		case tok == "F-0":
			ptn.Ops = append(ptn.Ops, &GameOver{common,
				tak.WinDetails{Winner: tak.White, Reason: tak.FlatsWin}})
		case tok == "0-F":
			ptn.Ops = append(ptn.Ops, &GameOver{common,
				tak.WinDetails{Winner: tak.Black, Reason: tak.FlatsWin}})
		case tok == "1/2-1/2":
			ptn.Ops = append(ptn.Ops, &GameOver{common,
				tak.WinDetails{Winner: tak.NoColor, Reason: tak.FlatsWin}})
		case tok == "1-0":
			ptn.Ops = append(ptn.Ops, &GameOver{common,
				tak.WinDetails{Winner: tak.White, Reason: tak.Resignation}})
		case tok == "0-1":
			ptn.Ops = append(ptn.Ops, &GameOver{common,
				tak.WinDetails{Winner: tak.Black, Reason: tak.Resignation}})
		default:
			trimmed := strings.TrimRight(tok, "?!'")
			move, e := ParseMove(trimmed)
			if e != nil {
				return e
			}
			ptn.Ops = append(ptn.Ops, &Move{common, move, tok[len(trimmed):]})
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

func (p *PTN) Render() string {
	var out bytes.Buffer
	for _, tag := range p.Tags {
		fmt.Fprintf(&out, "[%s \"%s\"]\n",
			tag.Name, strings.Replace(tag.Value, "\"", "", -1),
		)
	}
	out.WriteString("\n")

	for _, op := range p.Ops {
		switch o := op.(type) {
		case *MoveNumber:
			fmt.Fprintf(&out, "\n%d.", o.Number)
		case *Move:
			fmt.Fprintf(&out, " %s%s", FormatMove(&o.Move), o.Modifiers)
		case *Comment:
			fmt.Fprintf(&out, " {%s}", o.Comment)
		case *GameOver:
			var w string
			switch {
			case o.End.Reason == tak.FlatsWin && o.End.Winner == tak.Black:
				w = "0-F"
			case o.End.Reason == tak.FlatsWin && o.End.Winner == tak.White:
				w = "F-0"
			case o.End.Reason == tak.RoadWin && o.End.Winner == tak.Black:
				w = "0-R"
			case o.End.Reason == tak.RoadWin && o.End.Winner == tak.White:
				w = "R-0"
			case o.End.Reason == tak.Resignation && o.End.Winner == tak.Black:
				w = "0-1"
			case o.End.Reason == tak.Resignation && o.End.Winner == tak.White:
				w = "1-0"
			case o.End.Winner == tak.NoColor:
				w = "1/2-1/2"
			}
			fmt.Fprintf(&out, "\n%s\n", w)
		default:
		}
	}
	return out.String()
}
