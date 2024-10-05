package metrics

import (
	"os/exec"
	"strings"
)

func getOSVersion() (string, string) {
	cmd := exec.Command("sw_vers", "-productVersion")

	output, err := cmd.Output()
	if err != nil {
		return "", ""
	}

	return strings.TrimSpace(string(output)), ""
}
