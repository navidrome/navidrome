package jellyfin

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"iter"
	"strconv"

	"github.com/navidrome/navidrome/server/jellyfin/dto"
)

// streamItemsEnvelope writes a Jellyfin QueryResult as JSON, encoding items one at a time from a
// pull sequence instead of buffering the whole array — so a full-library /Items response streams from
// a DB cursor with peak memory of about one item. For a materialized slice the bytes are identical to
// json.NewEncoder(w).Encode(QueryResult{...}).
//
// A mid-stream error aborts without closing the envelope, leaving malformed JSON: the HTTP 200 is
// already committed, so a truncated-but-valid body would let a sync client treat the short list as
// complete (and prune local tracks). Malformed JSON forces the client's parser to fail and retry.
// Cursor-open errors are caught by the caller before any byte is written, so this only fires on a
// rare mid-iteration failure.
func streamItemsEnvelope(w io.Writer, items iter.Seq2[dto.BaseItemDto, error], total, start int) error {
	// bufio latches the first write error and returns it from Flush, so intermediate writes go unchecked.
	bw := bufio.NewWriterSize(w, 64*1024)
	write := func(s string) { _, _ = bw.WriteString(s) }
	// Reuse one buffer+encoder so per-item JSON doesn't allocate a fresh slice each time. The encoder
	// HTML-escapes like json.Marshal; the newline it appends is dropped below.
	var itemBuf bytes.Buffer
	enc := json.NewEncoder(&itemBuf)

	write(`{"Items":[`)
	first := true
	for item, err := range items {
		if err != nil {
			_ = bw.Flush()
			return err
		}
		if !first {
			write(",")
		}
		first = false
		itemBuf.Reset()
		if err := enc.Encode(item); err != nil {
			_ = bw.Flush()
			return err
		}
		b := itemBuf.Bytes()
		_, _ = bw.Write(b[:len(b)-1])
	}
	write(`],"TotalRecordCount":`)
	write(strconv.Itoa(total))
	write(`,"StartIndex":`)
	write(strconv.Itoa(start))
	write("}\n")
	return bw.Flush()
}

// streamQueryResult streams a fully materialized QueryResult (used by api.ok for the bounded,
// non-cursor responses). Callers always pass a non-nil Items slice, so an empty result renders as
// [] — identical to json.Encoder.
func streamQueryResult(w io.Writer, q dto.QueryResult) error {
	return streamItemsEnvelope(w, sliceItems(q.Items), q.TotalRecordCount, q.StartIndex)
}

// sliceItems adapts a slice to the pull sequence streamItemsEnvelope consumes.
func sliceItems(items []dto.BaseItemDto) iter.Seq2[dto.BaseItemDto, error] {
	return func(yield func(dto.BaseItemDto, error) bool) {
		for i := range items {
			if !yield(items[i], nil) {
				return
			}
		}
	}
}
