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
// A mid-stream error stops iteration but still closes the envelope (valid, if short, JSON) and is
// returned for logging; the HTTP status is already committed, as with a mid-encode failure before.
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
	var itemsErr error
	for item, err := range items {
		if err != nil {
			itemsErr = err
			break
		}
		if !first {
			write(",")
		}
		first = false
		itemBuf.Reset()
		if err := enc.Encode(item); err != nil {
			itemsErr = err
			break
		}
		b := itemBuf.Bytes()
		_, _ = bw.Write(b[:len(b)-1])
	}
	write(`],"TotalRecordCount":`)
	write(strconv.Itoa(total))
	write(`,"StartIndex":`)
	write(strconv.Itoa(start))
	write("}\n")
	if err := bw.Flush(); err != nil {
		return err
	}
	return itemsErr
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
