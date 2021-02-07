// +build !embed

package assets

import (
	"net/http"
	"sync"

	"github.com/navidrome/navidrome/log"
)

var once sync.Once

func AssetFile() http.FileSystem {
	once.Do(func() {
		log.Warn("Using external assets from 'ui/build' folder")
	})
	return http.Dir("ui/build")
}
