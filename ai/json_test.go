package ai

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestMarshalUnmarshal(t *testing.T) {
	cases := []struct {
		in  Weights
		out string
	}{
		{Weights{}, "{}"},
		{Weights{TopFlat: 100}, `{"TopFlat":100}`},
		{Weights{TopFlat: 100, Capstone: 150}, `{"Capstone":150,"TopFlat":100}`},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			out, e := json.Marshal(&tc.in)
			if e != nil {
				t.Fatalf("Marshal(): %v", e)
			}
			if string(out) != tc.out {
				t.Fatalf("Marshal() = %q != %q", out, tc.out)
			}

			var back Weights
			e = json.Unmarshal(out, &back)
			if e != nil {
				t.Fatalf("Unmarshal(%q): %v", out, e)
			}
			for i, v := range back {
				if tc.in[i] != v {
					t.Errorf("roundtrip[%d] = %v != %v", i, v, tc.in[i])
				}
			}
		})
	}
}
