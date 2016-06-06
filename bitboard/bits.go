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
	c.Size = size
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

func Flood(c *Constants, within uint64, seed uint64) uint64 {
	for {
		next := Grow(c, within, seed)
		if next == seed {
			return next
		}
		seed = next
	}
}

func Grow(c *Constants, within uint64, seed uint64) uint64 {
	next := seed
	next |= (seed << 1) &^ c.R
	next |= (seed >> 1) &^ c.L
	next |= (seed >> c.Size)
	next |= (seed << c.Size)
	return next & within
}

func FloodGroups(c *Constants, bits uint64, out []uint64) []uint64 {
	var seen uint64
	for bits != 0 {
		next := bits & (bits - 1)
		bit := bits &^ next

		if seen&bit == 0 {
			g := Flood(c, bits, bit)
			if g != bit {
				out = append(out, g)
			}
			seen |= g
		}

		bits = next
	}
	return out
}

func Dimensions(c *Constants, bits uint64) (w, h int) {
	if bits == 0 {
		return 0, 0
	}
	b := c.L
	for bits&b == 0 {
		b >>= 1
	}
	for b != 0 && bits&b != 0 {
		b >>= 1
		w++
	}
	b = c.T
	for bits&b == 0 {
		b >>= c.Size
	}
	for b != 0 && bits&b != 0 {
		b >>= c.Size
		h++
	}
	return w, h
}
