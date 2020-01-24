package engine

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/log"
)

// TODO Encapsulate as a io.Reader
func Stream(ctx context.Context, path string, bitRate int, maxBitRate int, w io.Writer) error {
	var f io.Reader
	var err error
	enabled := !conf.Sonic.DisableDownsampling
	if enabled && maxBitRate > 0 && bitRate > maxBitRate {
		f, err = downsample(ctx, path, maxBitRate)
	} else {
		f, err = os.Open(path)
	}
	if err != nil {
		log.Error(ctx, "Error opening file", "path", path, err)
		return err
	}
	if _, err = io.Copy(w, f); err != nil {
		log.Error(ctx, "Error copying file", "path", path, err)
		return err
	}
	return err
}

func downsample(ctx context.Context, path string, maxBitRate int) (f io.Reader, err error) {
	cmdLine, args := createDownsamplingCommand(path, maxBitRate)

	log.Debug(ctx, "Executing command", "cmdLine", cmdLine, "args", args)
	cmd := exec.Command(cmdLine, args...)
	cmd.Stderr = os.Stderr
	if f, err = cmd.StdoutPipe(); err != nil {
		return f, err
	}
	return f, cmd.Start()
}

func createDownsamplingCommand(path string, maxBitRate int) (string, []string) {
	cmd := conf.Sonic.DownsampleCommand

	split := strings.Split(cmd, " ")
	for i, s := range split {
		s = strings.Replace(s, "%s", path, -1)
		s = strings.Replace(s, "%b", strconv.Itoa(maxBitRate), -1)
		split[i] = s
	}

	return split[0], split[1:]
}
