package hasher

import (
	"hash/maphash"
	"strconv"
	"sync"

	"github.com/navidrome/navidrome/utils/random"
)

var instance = NewHasher()

func Reseed(id string) {
	instance.Reseed(id)
}

func SetSeed(id string, seed string) {
	instance.SetSeed(id, seed)
}

func HashFunc() func(id, str string) uint64 {
	return instance.HashFunc()
}

type Hasher struct {
	seeds    map[string]string
	mutex    sync.RWMutex
	hashSeed maphash.Seed
}

func NewHasher() *Hasher {
	h := new(Hasher)
	h.seeds = make(map[string]string)
	h.hashSeed = maphash.MakeSeed()
	return h
}

// SetSeed sets a seed for the given id
func (h *Hasher) SetSeed(id string, seed string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.seeds[id] = seed
}

// Reseed generates a new random seed for the given id
func (h *Hasher) Reseed(id string) {
	_ = h.reseed(id)
}

func (h *Hasher) reseed(id string) string {
	seed := strconv.FormatUint(random.Uint64(), 36)
	h.SetSeed(id, seed)
	return seed
}

// HashFunc returns a function that hashes a string using the seed for the given id
func (h *Hasher) HashFunc() func(id, str string) uint64 {
	return func(id, str string) uint64 {
		h.mutex.RLock()
		seed, ok := h.seeds[id]
		h.mutex.RUnlock()
		if !ok {
			seed = h.reseed(id)
		}

		return maphash.Bytes(h.hashSeed, []byte(seed+str))
	}
}
