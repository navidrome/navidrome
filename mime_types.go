package main

import "mime"

func init() {
	mt := map[string]string{
		".mp3":  "audio/mpeg",
		".ogg":  "audio/ogg",
		".oga":  "audio/ogg",
		".opus": "audio/ogg",
		".ogx":  "application/ogg",
		".aac":  "audio/mp4",
		".m4a":  "audio/mp4",
		".flac": "audio/flac",
		".wav":  "audio/x-wav",
		".wma":  "audio/x-ms-wma",
		".ape":  "audio/x-monkeys-audio",
		".mpc":  "audio/x-musepack",
		".shn":  "audio/x-shn",
		".flv":  "video/x-flv",
		".avi":  "video/avi",
		".mpg":  "video/mpeg",
		".mpeg": "video/mpeg",
		".mp4":  "video/mp4",
		".m4v":  "video/x-m4v",
		".mkv":  "video/x-matroska",
		".mov":  "video/quicktime",
		".wmv":  "video/x-ms-wmv",
		".ogv":  "video/ogg",
		".divx": "video/divx",
		".m2ts": "video/MP2T",
		".ts":   "video/MP2T",
		".webm": "video/webm",
		".gif":  "image/gif",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".bmp":  "image/bmp",
	}

	for ext, typ := range mt {
		mime.AddExtensionType(ext, typ)
	}
}
