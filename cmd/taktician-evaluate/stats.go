package main

import (
	"math"
	"math/big"
)

func binomprob(k, n int64, p float64) float64 {
	nk := big.NewFloat(0).SetInt(big.NewInt(0).Binomial(n, k))
	nk.Mul(nk, big.NewFloat(math.Pow(p, float64(k))))
	nk.Mul(nk, big.NewFloat(math.Pow(1-p, float64(n-k))))
	f, _ := nk.Float64()
	return f
}

func binomTest(succ, fail int64, p float64) float64 {
	var r float64
	for t := succ; t < (fail + succ); t++ {
		r += binomprob(t, succ+fail, p)
	}
	return r
}
