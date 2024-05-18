package mime

import (
	"mime"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/resources"
	"gopkg.in/yaml.v3"
)

type mimeConf struct {
	Types    map[string]string `yaml:"types"`
	Lossless []string          `yaml:"lossless"`
}

var LosslessFormats []string

func initMimeTypes() {
	// In some circumstances, Windows sets JS mime-type to `text/plain`!
	_ = mime.AddExtensionType(".js", "text/javascript")
	_ = mime.AddExtensionType(".css", "text/css")

	f, err := resources.FS().Open("mime_types.yaml")
	if err != nil {
		log.Fatal("Fatal error opening mime_types.yaml", err)
	}
	defer f.Close()

	var mimeConf mimeConf
	err = yaml.NewDecoder(f).Decode(&mimeConf)
	if err != nil {
		log.Fatal("Fatal error parsing mime_types.yaml", err)
	}
	for ext, typ := range mimeConf.Types {
		_ = mime.AddExtensionType(ext, typ)
	}

	for _, ext := range mimeConf.Lossless {
		LosslessFormats = append(LosslessFormats, strings.TrimPrefix(ext, "."))
	}
}

func init() {
	conf.AddHook(initMimeTypes)
}
