package tei

import (
	"strconv"
	"time"
)

type TimeControl struct {
	White time.Duration
	Black time.Duration
	WInc  time.Duration
	BInc  time.Duration
}

func formatTime(d time.Duration) string {
	ms := d / time.Millisecond
	if ms < 0 {
		ms = 0
	}
	return strconv.FormatUint(uint64(ms), 10)
}
