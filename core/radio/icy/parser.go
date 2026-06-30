package icy

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

var (
	ErrInvalidMetaInt      = errors.New("invalid icy metadata interval")
	ErrMissingTitleHandler = errors.New("missing icy title handler")
	errEndOfStream         = errors.New("icy stream ended")
)

// ReadStreamTitles reads ICY metadata blocks from r and emits changed StreamTitle values.
func ReadStreamTitles(ctx context.Context, r io.Reader, metaInt int, handleTitle func(string)) error {
	if metaInt <= 0 {
		return ErrInvalidMetaInt
	}
	if handleTitle == nil {
		return ErrMissingTitleHandler
	}

	audio := make([]byte, metaInt)
	length := make([]byte, 1)
	lastTitle := ""

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := readFullOrEnd(r, audio); err != nil {
			if errors.Is(err, errEndOfStream) {
				return nil
			}
			return err
		}

		if err := ctx.Err(); err != nil {
			return err
		}

		if err := readFullOrEnd(r, length); err != nil {
			if errors.Is(err, errEndOfStream) {
				return nil
			}
			return err
		}

		metadataLen := int(length[0]) * 16
		if metadataLen == 0 {
			continue
		}

		metadata := make([]byte, metadataLen)
		if err := readFullOrEnd(r, metadata); err != nil {
			if errors.Is(err, errEndOfStream) {
				return nil
			}
			return err
		}

		title := streamTitle(metadata)
		if title == "" || title == lastTitle {
			continue
		}

		lastTitle = title
		handleTitle(title)
	}
}

func readFullOrEnd(r io.Reader, buf []byte) error {
	if _, err := io.ReadFull(r, buf); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return errEndOfStream
		}
		return err
	}
	return nil
}

func streamTitle(metadata []byte) string {
	text := decodeMetadata(bytes.TrimRight(metadata, "\x00"))
	const prefix = "StreamTitle='"

	start := strings.Index(text, prefix)
	if start < 0 {
		return ""
	}

	start += len(prefix)
	end := strings.IndexByte(text[start:], '\'')
	if end < 0 {
		return ""
	}

	return strings.TrimSpace(text[start : start+end])
}

func decodeMetadata(metadata []byte) string {
	if utf8.Valid(metadata) {
		return string(metadata)
	}

	text, err := charmap.ISO8859_1.NewDecoder().String(string(metadata))
	if err != nil {
		return string(bytes.ToValidUTF8(metadata, nil))
	}
	return text
}
