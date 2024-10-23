package log

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"
)

func ShortDur(d time.Duration) string {
	var s string
	switch {
	case d > time.Hour:
		s = d.Round(time.Minute).String()
	case d > time.Minute:
		s = d.Round(time.Second).String()
	case d > time.Second:
		s = d.Round(10 * time.Millisecond).String()
	case d > time.Millisecond:
		s = d.Round(100 * time.Microsecond).String()
	default:
		s = d.String()
	}
	s = strings.TrimSuffix(s, "0s")
	return strings.TrimSuffix(s, "0m")
}

func StringerValue(s fmt.Stringer) string {
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Pointer && v.IsNil() {
		return "nil"
	}
	return s.String()
}

func CRLFWriter(w io.Writer) io.Writer {
	return &crlfWriter{w: w}
}

type crlfWriter struct {
	w        io.Writer
	lastByte byte
}

func (cw *crlfWriter) Write(p []byte) (int, error) {
	var written int
	for _, b := range p {
		if b == '\n' && cw.lastByte != '\r' {
			if _, err := cw.w.Write([]byte{'\r'}); err != nil {
				return written, err
			}
		}
		if _, err := cw.w.Write([]byte{b}); err != nil {
			return written, err
		}
		written++
		cw.lastByte = b
	}
	return written, nil
}
