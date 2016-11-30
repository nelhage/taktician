package ptn

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/nelhage/taktician/tak"
)

type Tag struct {
	Name  string
	Value string
}

type Op interface {
	op()
	clearSrc()

	Source() string
}

type opCommon struct {
	src string
}

func (o opCommon) Source() string {
	return o.src
}

func (o opCommon) op() {}

// for the tests
func (o *opCommon) clearSrc() {
	o.src = ""
}

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

type Result struct {
	opCommon
	Result string
}

func (r *Result) Winner() tak.Color {
	switch r.Result {
	case "R-0", "F-0", "1-0":
		return tak.White
	case "0-R", "0-F", "0-1":
		return tak.Black
	case "1/2-1/2":
		return tak.NoColor
	}
	return tak.NoColor
}

type PTN struct {
	Tags []Tag
	Ops  []Op
}

func ParsePTN(r io.Reader) (*PTN, error) {
	buf := bufio.NewReader(r)
	ch, _, err := buf.ReadRune()
	if err != nil {
		return nil, err
	}
	if ch != 0xFEFF {
		buf.UnreadRune()
	}
	var ptn PTN
	if err := readEvents(buf, &ptn); err != nil && err != io.EOF {
		return nil, err
	}
	if err := readMoves(buf, &ptn); err != nil && err != io.EOF {
		return nil, err
	}
	return &ptn, nil
}

func ParseFile(path string) (*PTN, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	return ParsePTN(f)
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

// PositionAtMove returns the position of the game after PTN move
// marker `move`, with `color` to play.
//
// `move=0` will cause the code to return the final position of the
// game.
func (p *PTN) PositionAtMove(move int, color tak.Color) (*tak.Position, error) {
	if color == tak.NoColor && move != 0 {
		return nil, fmt.Errorf("can't specify NoColor and move!=0")
	}
	it := p.Iterator()
	for it.Next() {
		if move > 0 && move == it.PTNMove() && it.Position().ToMove() == color {
			return it.Position(), nil
		}
	}
	if e := it.Err(); e != nil {
		return nil, e
	}

	if move > 0 {
		return nil, fmt.Errorf("move not found: %d", move)
	}
	return it.Position(), nil
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

var resultRE = regexp.MustCompile(`^(F|R|1/2|1|0)-(F|R|1/2|1|0)$`)

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
		case resultRE.MatchString(tok):
			ptn.Ops = append(ptn.Ops, &Result{common, tok})
		default:
			trimmed := strings.TrimRight(tok, "?!'")
			move, e := ParseMove(trimmed)
			if e != nil {
				return fmt.Errorf("bad move: %s", trimmed)
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
			fmt.Fprintf(&out, " %s%s", FormatMove(o.Move), o.Modifiers)
		case *Comment:
			fmt.Fprintf(&out, " {%s}", o.Comment)
		case *Result:
			fmt.Fprintf(&out, "\n%s\n", o.Result)
		default:
		}
	}
	out.WriteString("\n")
	return out.String()
}

func (p *PTN) AddMoves(moves []tak.Move) {
	for i, m := range moves {
		if i%2 == 0 {
			p.Ops = append(p.Ops, &MoveNumber{Number: i/2 + 1})
		}
		p.Ops = append(p.Ops, &Move{Move: m})
	}
}
