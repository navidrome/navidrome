package singleton

import (
	"fmt"
	"reflect"

	"github.com/navidrome/navidrome/log"
)

var (
	instances    = make(map[string]any)
	getOrCreateC = make(chan entry)
)

type entry struct {
	f       func() any
	object  any
	resultC chan any
}

// GetInstance returns an existing instance of object. If it is not yet created, calls `constructor`, stores the
// result for future calls and return it
func GetInstance[T any](constructor func() T) T {
	var t T
	e := entry{
		object: t,
		f: func() any {
			return constructor()
		},
		resultC: make(chan any),
	}
	getOrCreateC <- e
	v := <-e.resultC
	return v.(T)
}

func init() {
	go func() {
		for {
			e := <-getOrCreateC
			name := reflect.TypeOf(e.object).String()
			v, created := instances[name]
			if !created {
				v = e.f()
				log.Trace("Created new singleton", "type", name, "instance", fmt.Sprintf("%+v", v))
				instances[name] = v
			}
			e.resultC <- v
		}
	}()
}
