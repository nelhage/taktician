package tei

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCalcBudget(t *testing.T) {
	cases := []struct {
		Move   time.Duration
		Game   time.Duration
		Inc    time.Duration
		Expect time.Duration
	}{
		{0, 3 * time.Second, 3 * time.Second, 0},
		{time.Second, 3 * time.Second, 3 * time.Second, time.Second},
		{5 * time.Second, 3 * time.Second, 3 * time.Second, 0},
	}
	for _, tc := range cases {
		got := calcBudget(tc.Move, tc.Game, tc.Inc)
		if tc.Expect != 0 {
			assert.Equal(t, tc.Expect, got)
		}
		if tc.Game != 0 {
			assert.Less(t, int64(got), int64(tc.Game))
		}
		if tc.Move != 0 {
			assert.LessOrEqual(t, int64(got), int64(tc.Move))
		}
	}
}
