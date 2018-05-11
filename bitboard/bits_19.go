// +build go1.9

package bitboard

import "math/bits"

func Popcount(x uint64) int {
	return bits.OnesCount64(x)
}

func TrailingZeros(x uint64) uint {
	return uint(bits.TrailingZeros64(x))
}
