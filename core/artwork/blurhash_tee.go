package artwork

import (
	"bytes"
	"io"

	"github.com/navidrome/navidrome/utils/cache"
)

// teeReader mirrors bytes read from src into buf, and on Close invokes onComplete with the captured
// bytes only if the stream was fully consumed (EOF) and stayed within maxBytes. Partial reads and
// oversized streams are skipped, so a hash is only ever computed from a complete, bounded image.
type teeReader struct {
	src        io.ReadCloser
	buf        bytes.Buffer
	maxBytes   int
	onComplete func(data []byte)
	eof        bool
	over       bool
	done       bool
}

func newTeeReader(src io.ReadCloser, maxBytes int, onComplete func(data []byte)) *teeReader {
	return &teeReader{src: src, maxBytes: maxBytes, onComplete: onComplete}
}

func (t *teeReader) Read(p []byte) (int, error) {
	n, err := t.src.Read(p)
	if n > 0 && !t.over {
		if t.buf.Len()+n > t.maxBytes {
			t.over = true
			t.buf.Reset()
		} else {
			t.buf.Write(p[:n])
		}
	}
	if err == io.EOF {
		t.eof = true
	}
	return n, err
}

func (t *teeReader) Close() error {
	err := t.src.Close()
	if !t.done && t.eof && !t.over && t.onComplete != nil {
		t.done = true
		t.onComplete(t.buf.Bytes())
	}
	return err
}

// teeCachedStream wraps a *cache.CachedStream so reads are teed for blurhash capture while callers
// still see a ReadCloser. Seek is intentionally dropped: blurhash-eligible serves are full reads
// (every artwork handler does io.Copy), so no caller Seeks a teed stream.
type teeCachedStream struct {
	*cache.CachedStream
	tee *teeReader
}

func (t *teeCachedStream) Read(p []byte) (int, error) { return t.tee.Read(p) }
func (t *teeCachedStream) Close() error               { return t.tee.Close() }
