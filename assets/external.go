// +build !embed

package assets

import (
	"net/http"

	"github.com/cloudsonic/sonic-server/consts"
)

func AssetFile() http.FileSystem {
	return http.Dir(consts.UIAssetsLocalPath)
}
