package tak

import "math/rand"

const (
	fnvBasis = 14695981039346656037
	fnvPrime = 1099511628211
)

var basis [64]uint64

func init() {
	r := rand.New(rand.NewSource(0x7a3))
	for i := 0; i < 64; i++ {
		basis[i] = uint64(r.Int63())
	}
}

func hash8(basis uint64, b byte) uint64 {
	return (basis ^ uint64(b)) * fnvPrime
}

func hash64(basis uint64, w uint64) uint64 {
	h := basis
	h = (h ^ (w & 0xff)) * fnvPrime
	h = (h ^ ((w >> 8) & 0xff)) * fnvPrime
	h = (h ^ ((w >> 16) & 0xff)) * fnvPrime
	h = (h ^ (w >> 24)) * fnvPrime
	return h
}

func (p *Position) hashAt(i uint) uint64 {
	if p.Height[i] <= 1 {
		return 0
	}
	return hash64(hash8(basis[i], p.Height[i]), p.Stacks[i])
}

func (p *Position) Hash() uint64 {
	h := p.hash
	h = hash64(h, p.White)
	h = hash64(h, p.Black)
	h = hash64(h, p.Standing)
	h = hash64(h, p.Caps)
	h = hash8(h, byte(p.ToMove()))
	return h
}
