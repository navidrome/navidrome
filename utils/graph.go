package utils

import "github.com/karlkfi/inject"

var Graph inject.Graph

func init() {
	Graph = inject.NewGraph()
}
