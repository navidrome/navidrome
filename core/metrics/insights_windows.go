package metrics

import (
	"os/exec"
	"regexp"

	"golang.org/x/sys/windows"
)

// Ex: Microsoft Windows [Version 10.0.26100.1742]
var winVerRegex = regexp.MustCompile(`Microsoft Windows \[.+\s([\d\.]+)\]`)

func getOSVersion() (version string, _ string) {
	cmd := exec.Command("cmd", "/c", "ver")

	output, err := cmd.Output()
	if err != nil {
		return "", ""
	}

	matches := winVerRegex.FindStringSubmatch(string(output))
	if len(matches) != 2 {
		return string(output), ""
	}
	return matches[1], ""
}

func getFilesystemType(path string) (string, error) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}

	var volumeName, filesystemName [windows.MAX_PATH + 1]uint16
	var serialNumber uint32
	var maxComponentLen, filesystemFlags uint32

	err = windows.GetVolumeInformation(
		pathPtr,
		&volumeName[0],
		windows.MAX_PATH,
		&serialNumber,
		&maxComponentLen,
		&filesystemFlags,
		&filesystemName[0],
		windows.MAX_PATH)

	if err != nil {
		return "", err
	}

	return windows.UTF16ToString(filesystemName[:]), nil
}
