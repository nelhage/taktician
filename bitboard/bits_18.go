// +build !go1.9

package bitboard

func Popcount(x uint64) int {
	// bit population count, see
	// http://graphics.stanford.edu/~seander/bithacks.html#CountBitsSetParallel
	if x == 0 {
		return 0
	}
	x -= (x >> 1) & 0x5555555555555555
	x = (x>>2)&0x3333333333333333 + x&0x3333333333333333
	x += x >> 4
	x &= 0x0f0f0f0f0f0f0f0f
	x *= 0x0101010101010101
	return int(x >> 56)
}

func TrailingZeros(x uint64) uint {
	for i := uint(0); i < 64; i++ {
		if x&1<<i != 0 {
			return i
		}
	}
	return 64
}
