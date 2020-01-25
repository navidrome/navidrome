// +build !embed

package assets

import (
	"net/http"
	"sync"

	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/log"
)

var once sync.Once

func AssetFile() http.FileSystem {
	once.Do(func() {
		log.Warn("Using external assets from " + consts.UIAssetsLocalPath)
	})
	return http.Dir(consts.UIAssetsLocalPath)
}
