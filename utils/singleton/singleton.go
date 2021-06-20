package singleton

import (
	"reflect"
	"strings"

	"github.com/navidrome/navidrome/log"
)

var (
	instances    = make(map[string]interface{})
	getOrCreateC = make(chan *entry, 1)
)

type entry struct {
	constructor func() interface{}
	object      interface{}
	resultC     chan interface{}
}

// Get returns an existing instance of object. If it is not yet created, calls `constructor`, stores the
// result for future calls and return it
func Get(object interface{}, constructor func() interface{}) interface{} {
	e := &entry{
		constructor: constructor,
		object:      object,
		resultC:     make(chan interface{}),
	}
	getOrCreateC <- e
	return <-e.resultC
}

func init() {
	go func() {
		for {
			e := <-getOrCreateC
			name := reflect.TypeOf(e.object).String()
			name = strings.TrimPrefix(name, "*")
			v, created := instances[name]
			if !created {
				v = e.constructor()
				log.Trace("Created new singleton", "object", name, "instance", v)
				instances[name] = v
			}
			e.resultC <- v
		}
	}()
}
