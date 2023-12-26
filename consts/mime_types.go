package consts

import (
	"mime"
	"sort"
	"strings"
)

type format struct {
	typ      string
	lossless bool
}

var audioFormats = map[string]format{
	".mp3":  {typ: "audio/mpeg"},
	".ogg":  {typ: "audio/ogg"},
	".oga":  {typ: "audio/ogg"},
	".opus": {typ: "audio/ogg"},
	".aac":  {typ: "audio/mp4"},
	".alac": {typ: "audio/mp4", lossless: true},
	".m4a":  {typ: "audio/mp4"},
	".m4b":  {typ: "audio/mp4"},
	".flac": {typ: "audio/flac", lossless: true},
	".wav":  {typ: "audio/x-wav", lossless: true},
	".wma":  {typ: "audio/x-ms-wma"},
	".ape":  {typ: "audio/x-monkeys-audio", lossless: true},
	".mpc":  {typ: "audio/x-musepack"},
	".shn":  {typ: "audio/x-shn", lossless: true},
	".aif":  {typ: "audio/x-aiff"},
	".aiff": {typ: "audio/x-aiff"},
	".m3u":  {typ: "audio/x-mpegurl"},
	".pls":  {typ: "audio/x-scpls"},
	".dsf":  {typ: "audio/dsd", lossless: true},
	".wv":   {typ: "audio/x-wavpack", lossless: true},
	".wvp":  {typ: "audio/x-wavpack", lossless: true},
	".tak":  {typ: "audio/tak", lossless: true},
	".mka":  {typ: "audio/x-matroska"},
}
var imageFormats = map[string]string{
	".gif":  "image/gif",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".webp": "image/webp",
	".png":  "image/png",
	".bmp":  "image/bmp",
}

var LosslessFormats []string

func init() {
	for ext, fmt := range audioFormats {
		_ = mime.AddExtensionType(ext, fmt.typ)
		if fmt.lossless {
			LosslessFormats = append(LosslessFormats, strings.TrimPrefix(ext, "."))
		}
	}
	sort.Strings(LosslessFormats)
	for ext, typ := range imageFormats {
		_ = mime.AddExtensionType(ext, typ)
	}

	// In some circumstances, Windows sets JS mime-type to `text/plain`!
	_ = mime.AddExtensionType(".js", "text/javascript")
	_ = mime.AddExtensionType(".css", "text/css")
}
