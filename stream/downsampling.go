package stream

import (
	"github.com/astaxie/beego"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func Stream(path string, bitRate int, maxBitRate int, w io.Writer) error {
	if maxBitRate > 0 && bitRate > maxBitRate {
		cmdLine, args := createDownsamplingCommand(path, maxBitRate)

		beego.Debug("Executing cmd:", cmdLine, args)
		cmd := exec.Command(cmdLine, args...)
		cmd.Stderr = os.Stderr
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			beego.Error("Error executing", cmdLine, ":", err)
			return err
		}
		if err = cmd.Start(); err != nil {
			beego.Error("Error executing", cmdLine, ":", err)
		} else {
			_, err = io.Copy(w, stdout)
		}
		return err
	} else {
		f, err := os.Open(path)
		if err != nil {
			beego.Error("Error opening file", path, ":", err)
			return err
		}
		_, err = io.Copy(w, f)
		return err
	}
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
