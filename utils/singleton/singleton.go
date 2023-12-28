package singleton

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/navidrome/navidrome/log"
)

var (
	instances = make(map[string]any)
	lock      sync.RWMutex
)

// GetInstance returns an existing instance of object. If it is not yet created, calls `constructor`, stores the
// result for future calls and returns it
func GetInstance[T any](constructor func() T) T {
	var v T
	name := reflect.TypeOf(v).String()

	v, available := func() (T, bool) {
		lock.RLock()
		defer lock.RUnlock()
		v, available := instances[name].(T)
		return v, available
	}()

	if available {
		return v
	}

	lock.Lock()
	defer lock.Unlock()
	v, available = instances[name].(T)
	if available {
		return v
	}

	v = constructor()
	log.Trace("Created new singleton", "type", name, "instance", fmt.Sprintf("%+v", v))
	instances[name] = v
	return v
}
