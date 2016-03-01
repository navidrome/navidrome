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
	if definitions[typ] == nil {
		Graph.Define(ptr, inject.NewProvider(constructor, args...))
	} else {
		Graph.Define(definitions[typ], inject.NewProvider(constructor, args...))
	}
	definitions[typ] = ptr
}

func init() {
	definitions = make(map[reflect.Type]interface{})
	Graph = inject.NewGraph()
}
