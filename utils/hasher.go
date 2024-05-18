package utils

import "hash/maphash"

var Hasher = newHasher()

type hasher struct {
	seeds map[string]maphash.Seed
}

func newHasher() *hasher {
	h := new(hasher)
	h.seeds = make(map[string]maphash.Seed)
	return h
}

func (h *hasher) Reseed(id string) {
	h.seeds[id] = maphash.MakeSeed()
}

func (h *hasher) HashFunc() func(id, str string) uint64 {
	return func(id, str string) uint64 {
		var hash maphash.Hash
		var seed maphash.Seed
		var ok bool
		if seed, ok = h.seeds[id]; !ok {
			seed = maphash.MakeSeed()
			h.seeds[id] = seed
		}
		hash.SetSeed(seed)
		_, _ = hash.WriteString(str)
		return hash.Sum64()
	}
}
