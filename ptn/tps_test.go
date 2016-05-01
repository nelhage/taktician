package ptn

import (
	"reflect"
	"testing"

	"nelhage.com/tak/tak"
)

func TestParseTPS(t *testing.T) {
	tpn := `x3,12,2S/x,22S,22C,11,21/121,212,12,1121C,1212S/21S,1,21,211S,12S/x,21S,2,x2 1 26`
	p, e := ParseTPS(tpn)
	if e != nil {
		t.Fatal("parse error", e)
	}
	if p.Size() != 5 {
		t.Error("size=", p.Size())
	}
	if p.MoveNumber() != 50 {
		t.Error("move=", p.MoveNumber())
	}
	expect := [][]tak.Square{
		//y=0
		[]tak.Square{
			nil,
			tak.Square{
				tak.MakePiece(tak.White, tak.Standing),
				tak.MakePiece(tak.Black, tak.Flat)},
			tak.Square{
				tak.MakePiece(tak.Black, tak.Flat)},
			nil,
			nil,
		},
		//y=1
		[]tak.Square{
			tak.Square{
				tak.MakePiece(tak.White, tak.Standing),
				tak.MakePiece(tak.Black, tak.Flat)},
			tak.Square{
				tak.MakePiece(tak.White, tak.Flat)},
			tak.Square{
				tak.MakePiece(tak.White, tak.Flat),
				tak.MakePiece(tak.Black, tak.Flat)},
			tak.Square{
				tak.MakePiece(tak.White, tak.Standing),
				tak.MakePiece(tak.White, tak.Flat),
				tak.MakePiece(tak.Black, tak.Flat)},
			tak.Square{
				tak.MakePiece(tak.Black, tak.Standing),
				tak.MakePiece(tak.White, tak.Flat)},
		},
		//y=2
		[]tak.Square{
			tak.Square{
				tak.MakePiece(tak.White, tak.Flat),
				tak.MakePiece(tak.Black, tak.Flat),
				tak.MakePiece(tak.White, tak.Flat)},
			tak.Square{
				tak.MakePiece(tak.Black, tak.Flat),
				tak.MakePiece(tak.White, tak.Flat),
				tak.MakePiece(tak.Black, tak.Flat)},
			tak.Square{
				tak.MakePiece(tak.Black, tak.Flat),
				tak.MakePiece(tak.White, tak.Flat)},
			tak.Square{
				tak.MakePiece(tak.White, tak.Capstone),
				tak.MakePiece(tak.Black, tak.Flat),
				tak.MakePiece(tak.White, tak.Flat),
				tak.MakePiece(tak.White, tak.Flat)},
			tak.Square{
				tak.MakePiece(tak.Black, tak.Standing),
				tak.MakePiece(tak.White, tak.Flat),
				tak.MakePiece(tak.Black, tak.Flat),
				tak.MakePiece(tak.White, tak.Flat)},
		},
		//y=3
		[]tak.Square{
			nil,
			tak.Square{
				tak.MakePiece(tak.Black, tak.Standing),
				tak.MakePiece(tak.Black, tak.Flat)},
			tak.Square{
				tak.MakePiece(tak.Black, tak.Capstone),
				tak.MakePiece(tak.Black, tak.Flat)},
			tak.Square{
				tak.MakePiece(tak.White, tak.Flat),
				tak.MakePiece(tak.White, tak.Flat)},
			tak.Square{
				tak.MakePiece(tak.White, tak.Flat),
				tak.MakePiece(tak.Black, tak.Flat)},
		},
		//y=4
		[]tak.Square{
			nil,
			nil,
			nil,
			tak.Square{
				tak.MakePiece(tak.Black, tak.Flat),
				tak.MakePiece(tak.White, tak.Flat)},
			tak.Square{
				tak.MakePiece(tak.Black, tak.Standing),
			},
		},
	}
	for y, row := range expect {
		for x, want := range row {
			got := p.At(x, y)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%d,%d: got=%#v want=%#v", x, y, got, want)
			}
		}
	}
}

func TestRenderTPS(t *testing.T) {
	tps := `x3,12,2S/x,22S,22C,11,21/121,212,12,1121C,1212S/21S,1,21,211S,12S/x,21S,2,x2 1 26`
	p, e := ParseTPS(tps)
	if e != nil {
		t.Fatal("parse error", e)
	}
	if p == nil {
		t.Fatal("parse returned nil?")
	}
	out := FormatTPS(p)
	if out != tps {
		t.Fatalf("FormatTPS:\n in= `%s`\n out=`%s`", tps, out)
	}
}
