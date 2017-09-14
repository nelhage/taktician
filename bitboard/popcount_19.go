// +build go.19

package bitboard

import "math/bits"

func Popcount(x uint64) int {
	return bits.OnesCount64(x)
}
