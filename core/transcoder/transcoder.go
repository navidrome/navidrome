package transcoder

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils/cache/item"
)

type CacheInvalidator interface {
	Invalidate(ctx context.Context, arg item.Item) error
}

type Invalidator struct {
	CacheInvalidator
	item.Item
}

type TranscoderWaiter struct {
	ReachedEOF chan struct{}
	io.ReadCloser
}

type Transcoder interface {
	Start(ctx context.Context, command, path string, maxBitRate int, invalidator Invalidator) (resp TranscoderWaiter, err error)
}

func New() Transcoder {
	return &externalTranscoder{}
}

type externalTranscoder struct{}

func (e *externalTranscoder) Start(ctx context.Context, command, path string, maxBitRate int, invalidator Invalidator) (resp TranscoderWaiter, err error) {
	args := createTranscodeCommand(command, path, maxBitRate)

	log.Trace(ctx, "Executing transcoding command", "cmd", args)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...) // #nosec
	cmd.Stderr = os.Stderr
	f, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	if err = cmd.Start(); err != nil {
		return
	}

	resp.ReachedEOF = make(chan struct{})
	resp.ReadCloser = f

	go func() {
		<-resp.ReachedEOF
		err := cmd.Wait()
		if err != nil {
			// Avoid cache poisoning. Assume all errs are invalid files
			if invalidator.CacheInvalidator != nil && invalidator.Item != nil {
				_ = invalidator.Invalidate(ctx, invalidator.Item)
			}

			log.Trace(ctx, "Error while waiting for transcode finish", err)
		}

		_ = resp.Close()
	}()

	return
}

// Path will always be an absolute path
func createTranscodeCommand(cmd, path string, maxBitRate int) []string {
	split := strings.Split(cmd, " ")
	for i, s := range split {
		s = strings.ReplaceAll(s, "%s", path)
		s = strings.ReplaceAll(s, "%b", strconv.Itoa(maxBitRate))
		split[i] = s
	}

	return split
}
