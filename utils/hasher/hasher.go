package hasher

import "hash/maphash"

var instance = NewHasher()

func Reseed(id string) {
	instance.Reseed(id)
}

func HashFunc() func(id, str string) uint64 {
	return instance.HashFunc()
}

type hasher struct {
	seeds map[string]maphash.Seed
}

func NewHasher() *hasher {
	h := new(hasher)
	h.seeds = make(map[string]maphash.Seed)
	return h
}

// Reseed generates a new seed for the given id
func (h *hasher) Reseed(id string) {
	h.seeds[id] = maphash.MakeSeed()
}

// HashFunc returns a function that hashes a string using the seed for the given id
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
