package metrics

import (
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
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

// MountInfo represents an entry from /proc/self/mountinfo
type MountInfo struct {
	MountPoint string
	FSType     string
}

var fsTypeMap = map[int64]string{
	// Add filesystem type mappings
	0x9123683E: "btrfs",
	0x0000EF53: "ext2/ext3/ext4",
	0x00006969: "nfs",
	0x58465342: "xfs",
	0x2FC12FC1: "zfs",
	0x01021994: "tmpfs",
	0x28cd3d45: "cramfs",
	0x64626720: "debugfs",
	0x73717368: "squashfs",
	0x62656572: "sysfs",
	0x9fa0:     "proc",
	0x61756673: "aufs",
	0x794c7630: "overlayfs",
	0x6a656a63: "fakeowner", // FS inside a container
	// Include other filesystem types as needed
}

func getFilesystemType(path string) (string, error) {
	var fsStat syscall.Statfs_t
	err := syscall.Statfs(path, &fsStat)
	if err != nil {
		return "", err
	}

	fsType := fsStat.Type

	fsName, exists := fsTypeMap[int64(fsType)]
	if !exists {
		fsName = fmt.Sprintf("unknown(0x%x)", fsType)
	}

	return fsName, nil
}
