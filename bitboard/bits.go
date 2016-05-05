package bitboard

type Constants struct {
	Size       uint
	L, R, T, B uint64
	Mask       uint64
}

func Precompute(size uint) Constants {
	var c Constants
	for i := uint(0); i < size; i++ {
		c.R |= 1 << (i * size)
	}
	c.L = c.R << (size - 1)
	c.T = ((1 << size) - 1) << (size * (size - 1))
	c.B = (1 << size) - 1
	c.Mask = 1<<(size*size) - 1
	return c
}

func Popcount(x uint64) int {
	// bit population count, see
	// http://graphics.stanford.edu/~seander/bithacks.html#CountBitsSetParallel
	x -= (x >> 1) & 0x5555555555555555
	x = (x>>2)&0x3333333333333333 + x&0x3333333333333333
	x += x >> 4
	x &= 0x0f0f0f0f0f0f0f0f
	x *= 0x0101010101010101
	return int(x >> 56)
}
