package metrics

import (
	"os/exec"
	"strings"
	"syscall"
)

func getOSVersion() (string, string) {
	cmd := exec.Command("sw_vers", "-productVersion")

	output, err := cmd.Output()
	if err != nil {
		return "", ""
	}

	return strings.TrimSpace(string(output)), ""
}

func getFilesystemType(path string) (string, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return "", err
	}

	// Convert the filesystem type name from [16]int8 to string
	fsType := make([]byte, 0, 16)
	for _, c := range stat.Fstypename {
		if c == 0 {
			break
		}
		fsType = append(fsType, byte(c))
	}

	return string(fsType), nil
}
