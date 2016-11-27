package tak

import (
	"reflect"
	"testing"
)

func TestMkSlides(t *testing.T) {
	cases := []struct {
		out uint32
		in  []int
	}{
		{
			0,
			nil,
		},
		{
			0x1,
			[]int{1},
		},
		{
			0x321,
			[]int{1, 2, 3},
		},
	}

	for _, tc := range cases {
		s := MkSlides(tc.in...)
		if uint32(s) != tc.out {
			t.Errorf("%v: got %x != %x", tc.in, s, tc.out)
		}

		var out []int
		if !s.Empty() {
			for it, ok := s.Iterator(); ok; it, ok = it.Next() {
				out = append(out, it.Elem())
			}
		}
		if !reflect.DeepEqual(out, tc.in) {
			t.Errorf("rt(%v) = %v", tc.in, out)
		}
	}
}
