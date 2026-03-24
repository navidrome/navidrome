package ioutils

import (
	"io"
	"os"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// UTF8Reader wraps an io.Reader to handle Byte Order Mark (BOM) properly.
// It strips UTF-8 BOM if present, and converts UTF-16 (LE/BE) to UTF-8.
// This is particularly useful for reading user-provided text files (like LRC lyrics,
// playlists) that may have been created on Windows, which often adds BOM markers.
//
// Reference: https://en.wikipedia.org/wiki/Byte_order_mark
func UTF8Reader(r io.Reader) io.Reader {
	return transform.NewReader(r, unicode.BOMOverride(unicode.UTF8.NewDecoder()))
}

// UTF8ReadFile reads the named file and returns its contents as a byte slice,
// automatically handling BOM markers. It's similar to os.ReadFile but strips
// UTF-8 BOM and converts UTF-16 encoded files to UTF-8.
func UTF8ReadFile(filename string) ([]byte, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := UTF8Reader(file)
	return io.ReadAll(reader)
}
