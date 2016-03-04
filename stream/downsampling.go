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
		cmd := exec.Command(cmdLine, args...)
		beego.Debug("Executing cmd:", cmdLine, args)

		cmd.Stdout = w
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			beego.Error("Error executing", cmdLine, ":", err)
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
