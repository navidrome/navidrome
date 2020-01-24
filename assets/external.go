// +build !embed

package assets

import (
	"net/http"

	"github.com/deluan/navidrome/consts"
)

func AssetFile() http.FileSystem {
	return http.Dir(consts.UIAssetsLocalPath)
}
