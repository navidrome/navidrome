package consts

import "mime"

func init() {
	mt := map[string]string{
		".mp3":  "audio/mpeg",
		".ogg":  "audio/ogg",
		".oga":  "audio/ogg",
		".opus": "audio/ogg",
		".aac":  "audio/mp4",
		".m4a":  "audio/mp4",
		".m4b":  "audio/mp4",
		".flac": "audio/flac",
		".wav":  "audio/x-wav",
		".wma":  "audio/x-ms-wma",
		".ape":  "audio/x-monkeys-audio",
		".mpc":  "audio/x-musepack",
		".shn":  "audio/x-shn",
		".aif":  "audio/x-aiff",
		".aiff": "audio/x-aiff",
		".m3u":  "audio/x-mpegurl",
		".pls":  "audio/x-scpls",
		".dsf":  "audio/dsd",
		".gif":  "image/gif",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".bmp":  "image/bmp",
	}

	for ext, typ := range mt {
		_ = mime.AddExtensionType(ext, typ)
	}
}
