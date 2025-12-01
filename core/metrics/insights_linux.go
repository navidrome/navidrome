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
			version = strings.ReplaceAll(strings.TrimPrefix(line, "VERSION_ID="), "\"", "")
		}
		if strings.HasPrefix(line, "ID=") {
			distro = strings.ReplaceAll(strings.TrimPrefix(line, "ID="), "\"", "")
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
	0x5346414f: "afs",
	0x187:      "autofs",
	0x61756673: "aufs",
	0x9123683E: "btrfs",
	0xc36400:   "ceph",
	0xff534d42: "cifs",
	0x28cd3d45: "cramfs",
	0x64626720: "debugfs",
	0xf15f:     "ecryptfs",
	0x2011bab0: "exfat",
	0x0000EF53: "ext2/ext3/ext4",
	0xf2f52010: "f2fs",
	0x6a656a63: "fakeowner", // FS inside a container
	0x65735546: "fuse",
	0x4244:     "hfs",
	0x482b:     "hfs+",
	0x9660:     "iso9660",
	0x3153464a: "jfs",
	0x00006969: "nfs",
	0x5346544e: "ntfs", // NTFS_SB_MAGIC
	0x7366746e: "ntfs",
	0x794c7630: "overlayfs",
	0x9fa0:     "proc",
	0x517b:     "smb",
	0xfe534d42: "smb2",
	0x73717368: "squashfs",
	0x62656572: "sysfs",
	0x01021994: "tmpfs",
	0x01021997: "v9fs",
	0x786f4256: "vboxsf",
	0x4d44:     "vfat",
	0xca451a4e: "virtiofs",
	0x58465342: "xfs",
	0x2FC12FC1: "zfs",

	// Signed/unsigned conversion issues (negative hex values converted to uint32)
	-0x6edc97c2: "btrfs", // 0x9123683e
	-0x1acb2be:  "smb2",  // 0xfe534d42
	-0xacb2be:   "cifs",  // 0xff534d42
	-0xd0adff0:  "f2fs",  // 0xf2f52010
}

func getFilesystemType(path string) (string, error) {
	var fsStat syscall.Statfs_t
	err := syscall.Statfs(path, &fsStat)
	if err != nil {
		return "", err
	}

	fsType := fsStat.Type

	fsName, exists := fsTypeMap[int64(fsType)] //nolint:unconvert
	if !exists {
		fsName = fmt.Sprintf("unknown(0x%x)", fsType)
	}

	return fsName, nil
}
