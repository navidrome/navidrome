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
	bw := bufio.NewWriterSize(w, 64*1024)
	_, _ = bw.WriteString(`{"Items":[`)
	if err := encodeItems(bw, items); err != nil {
		_ = bw.Flush()
		return err
	}
	_, _ = bw.WriteString(`],"TotalRecordCount":`)
	_, _ = bw.WriteString(strconv.Itoa(total))
	_, _ = bw.WriteString(`,"StartIndex":`)
	_, _ = bw.WriteString(strconv.Itoa(start))
	_, _ = bw.WriteString("}\n")
	return bw.Flush()
}

// streamItemsArray writes items as a bare JSON array — the shape /Items/Latest returns, which has no
// QueryResult envelope. Same streaming and mid-stream-error behavior as streamItemsEnvelope.
func streamItemsArray(w io.Writer, items iter.Seq2[dto.BaseItemDto, error]) error {
	bw := bufio.NewWriterSize(w, 64*1024)
	_, _ = bw.WriteString("[")
	if err := encodeItems(bw, items); err != nil {
		_ = bw.Flush()
		return err
	}
	_, _ = bw.WriteString("]\n")
	return bw.Flush()
}

// encodeItems writes items comma-separated, encoding one at a time. bufio latches the first write
// error and returns it from Flush, so intermediate writes go unchecked.
func encodeItems(bw *bufio.Writer, items iter.Seq2[dto.BaseItemDto, error]) error {
	// Reuse one buffer+encoder so per-item JSON doesn't allocate a fresh slice each time. The encoder
	// HTML-escapes like json.Marshal; the newline it appends is dropped below.
	var itemBuf bytes.Buffer
	enc := json.NewEncoder(&itemBuf)
	first := true
	for item, err := range items {
		if err != nil {
			return err
		}
		if !first {
			_, _ = bw.WriteString(",")
		}
		first = false
		itemBuf.Reset()
		if err := enc.Encode(item); err != nil {
			return err
		}
		b := itemBuf.Bytes()
		_, _ = bw.Write(b[:len(b)-1])
	}
	return nil
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
