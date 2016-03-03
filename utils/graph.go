package utils

import (
	"github.com/karlkfi/inject"
	"reflect"
)

var Graph inject.Graph

var (
	definitions map[reflect.Type]interface{}
)

func DefineSingleton(ptr interface{}, constructor interface{}, args ...interface{}) {
	typ := reflect.TypeOf(ptr)
	provider := inject.NewProvider(constructor, args...)

	if _, found := definitions[typ]; found {
		ptr = definitions[typ]
	} else {
		definitions[typ] = ptr
	}
	Graph.Define(ptr, provider)
}

func init() {
	definitions = make(map[reflect.Type]interface{})
	Graph = inject.NewGraph()
}
