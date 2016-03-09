package engine

import (
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/astaxie/beego"
)

// TODO Encapsulate as a io.Reader
func Stream(path string, bitRate int, maxBitRate int, w io.Writer) error {
	var f io.Reader
	var err error
	if maxBitRate > 0 && bitRate > maxBitRate {
		f, err = downsample(path, maxBitRate)
	} else {
		f, err = os.Open(path)
	}
	if err != nil {
		beego.Error("Error opening file", path, ":", err)
		return err
	}
	if _, err = io.Copy(w, f); err != nil {
		beego.Error("Error copying file", path, ":", err)
		return err
	}
	return err
}

func downsample(path string, maxBitRate int) (f io.Reader, err error) {
	cmdLine, args := createDownsamplingCommand(path, maxBitRate)

	beego.Debug("Executing cmd:", cmdLine, args)
	cmd := exec.Command(cmdLine, args...)
	cmd.Stderr = os.Stderr
	if f, err = cmd.StdoutPipe(); err != nil {
		return f, err
	}
	return f, cmd.Start()
}

func createDownsamplingCommand(path string, maxBitRate int) (string, []string) {
	cmd := beego.AppConfig.String("downsampleCommand")

	split := strings.Split(cmd, " ")
	for i, s := range split {
		s = strings.Replace(s, "%s", path, -1)
		s = strings.Replace(s, "%b", strconv.Itoa(maxBitRate), -1)
		split[i] = s
	}

	return split[0], split[1:len(split)]
}
