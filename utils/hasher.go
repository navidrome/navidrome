package utils

import "hash/maphash"

var Hasher = newHasher()

type hasher struct {
	seed maphash.Seed
}

func newHasher() *hasher {
	h := new(hasher)
	h.Reseed()
	return h
}

func (h *hasher) Reseed() {
	h.seed = maphash.MakeSeed()
}

func (h *hasher) HashFunc() func(str string) uint64 {
	return func(str string) uint64 {
		var hash maphash.Hash
		hash.SetSeed(h.seed)
		_, _ = hash.WriteString(str)
		return hash.Sum64()
	}
}
