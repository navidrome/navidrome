package metrics

import (
	"io"
	"os"
	"strings"
)

func getOSVersion() (string, string) {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return "", ""
	}
	defer file.Close()

	osRelease, err := io.ReadAll(file)
	if err != nil {
		return "", ""
	}

	lines := strings.Split(string(osRelease), "\n")
	version := ""
	distro := ""
	for _, line := range lines {
		if strings.HasPrefix(line, "VERSION_ID=") {
			version = strings.ReplaceAll(strings.Trim(line, "VERSION_ID="), "\"", "")
		}
		if strings.HasPrefix(line, "ID=") {
			distro = strings.ReplaceAll(strings.Trim(line, "ID="), "\"", "")
		}
	}
	return version, distro
}
