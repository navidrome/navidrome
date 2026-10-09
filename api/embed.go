package api

import (
	_ "embed"
	"encoding/json"
	"sync"
)

//go:embed bundled/openapi.json
var specJSON []byte

//go:embed bundled/openapi.yaml
var specYAML []byte

func SpecJSON() []byte {
	return specJSON
}

func SpecYAML() []byte {
	return specYAML
}

var specVersion = sync.OnceValue(func() string {
	var doc struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
	}
	_ = json.Unmarshal(SpecJSON(), &doc)
	return doc.Info.Version
})

func SpecVersion() string {
	return specVersion()
}
