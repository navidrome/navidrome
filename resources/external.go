// +build !embed

package resources

import (
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/navidrome/navidrome/log"
)

var once sync.Once

func Asset(filePath string) ([]byte, error) {
	f, err := AssetFile().Open(filePath)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(f)
}

func AssetFile() http.FileSystem {
	once.Do(func() {
		log.Warn("Using external resources from 'resources' folder")
	})
	return http.Dir("resources")
}
