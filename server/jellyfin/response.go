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

// streamItemsEnvelope writes a QueryResult, byte-identical to json.NewEncoder(w).Encode(q).
//
// A mid-stream error aborts without closing the envelope: the 200 is already committed, so a
// truncated-but-valid body would let a sync client treat the short list as the whole library and
// prune local tracks. Malformed JSON forces its parser to fail instead. Callers open the cursor
// before the first byte, so this only fires on a rare mid-iteration failure.
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

// streamItemsArray writes a bare JSON array — the shape /Items/Latest returns, with no envelope.
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

// encodeItems writes items comma-separated. Unlike the fixed envelope writes, these are checked:
// bufio surfaces a latched write error here once a flush fails, and a client that has gone away must
// abandon the scan rather than pull the rest of the library through the cursor — which would hold its
// pooled DB connection and stream slot for a response nobody is reading.
func encodeItems(bw *bufio.Writer, items iter.Seq2[dto.BaseItemDto, error]) error {
	// One reused buffer+encoder, so per-item JSON doesn't allocate. Encode HTML-escapes like
	// json.Marshal, and appends a newline that's dropped below.
	var itemBuf bytes.Buffer
	enc := json.NewEncoder(&itemBuf)
	first := true
	for item, err := range items {
		if err != nil {
			return err
		}
		if !first {
			if _, err := bw.WriteString(","); err != nil {
				return err
			}
		}
		first = false
		itemBuf.Reset()
		if err := enc.Encode(item); err != nil {
			return err
		}
		b := itemBuf.Bytes()
		if _, err := bw.Write(b[:len(b)-1]); err != nil {
			return err
		}
	}
	return nil
}

func sliceItems(items []dto.BaseItemDto) iter.Seq2[dto.BaseItemDto, error] {
	return func(yield func(dto.BaseItemDto, error) bool) {
		for i := range items {
			if !yield(items[i], nil) {
				return
			}
		}
	}
}
