package ffmpeg

import (
	"errors"
	"io"
	"os"
	"strings"

	"github.com/kdomanski/iso9660"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

var ErrorNotAnISO = errors.New("not iso file")

type isoWVReader struct {
	f *os.File
	i *iso9660.Image
	r io.Reader
}

func (r *isoWVReader) Read(p []byte) (int, error) {
	if r.r != nil {
		return r.r.Read(p)
	}
	return 0, io.ErrClosedPipe
}

func (r *isoWVReader) Close() error {
	if r.f != nil {
		return r.f.Close()
	}

	return nil
}

func isISO(f *os.File) bool {
	_, err := f.Seek(32769, 0)
	if err != nil {
		return false
	}
	buf := make([]byte, 5)
	count, err := f.Read(buf)
	if err != nil {
		return false
	}
	if count != 5 {
		return false
	}
	if !(buf[0] == 0x43 && buf[1] == 0x44 &&
		buf[2] == 0x30 && buf[3] == 0x30 &&
		buf[4] == 0x31) {
		return false
	}
	_, err = f.Seek(0, 0)
	return err == nil
}

func (r *isoWVReader) open(path string) error {
	var err error
	r.f, err = os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = r.f.Close()
			r.f = nil
		}
	}()

	if !isISO(r.f) {
		return ErrorNotAnISO
	}

	r.i, err = iso9660.OpenImage(r.f)
	if err != nil {
		return err
	}
	root, err := r.i.RootDir()
	if err != nil {
		return err
	}
	children, err := root.GetChildren()
	if err != nil {
		return err
	}

	for _, entry := range children {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".WV") {
			continue
		}
		r.r = entry.Reader()
		return nil
	}

	return nil
}

func openWV(path string) io.ReadCloser {
	reader := isoWVReader{}

	if err := reader.open(path); err != nil {
		if errors.Is(err, ErrorNotAnISO) {
			log.Trace("Can't open ISO image file", "error", err)
		}
		return nil
	}

	return &reader
}

func openWVFromISOMedia(mf *model.MediaFile) io.ReadCloser {
	if !isSubTrack(mf) {
		return nil
	}

	return openWVFromISO(mf.Path)
}

func openWVFromISO(path string) io.ReadCloser {
	if !(strings.HasSuffix(path, ".wv") || strings.HasSuffix(path, ".iso.wv")) {
		return nil
	}

	return openWV(path)
}
