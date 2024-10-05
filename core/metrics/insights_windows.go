package metrics

import (
	"os/exec"
	"regexp"
)

// Ex: Microsoft Windows [Version 10.0.26100.1742]
var winVerRegex = regexp.MustCompile(`Microsoft Windows \[Version ([\d\.]+)\]`)

func getOSVersion() (version string, _ string) {
	cmd := exec.Command("cmd", "/c", "ver")

	output, err := cmd.Output()
	if err != nil {
		return "", ""
	}

	matches := winVerRegex.FindStringSubmatch(string(output))
	if len(matches) != 2 {
		return output, ""
	}
	return matches[1], ""
}
