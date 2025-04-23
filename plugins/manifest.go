package plugins

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
)

type PluginManifest struct {
	Services []string `json:"services"`
}

// LoadManifest loads and parses the manifest.json file from the given plugin directory.
func LoadManifest(pluginDir string) (*PluginManifest, error) {
	manifestPath := filepath.Join(pluginDir, "manifest.json")
	data, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	var manifest PluginManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}
