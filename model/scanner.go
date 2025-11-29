package model

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ScanTarget represents a specific folder within a library to be scanned.
// NOTE: This struct is used as a map key, so it should only contain comparable types.
type ScanTarget struct {
	LibraryID  int
	FolderPath string // Relative path within the library, or "" for entire library
}

func (st ScanTarget) String() string {
	return fmt.Sprintf("%d:%s", st.LibraryID, st.FolderPath)
}

// ScannerStatus holds information about the current scan status
type ScannerStatus struct {
	Scanning    bool
	LastScan    time.Time
	Count       uint32
	FolderCount uint32
	LastError   string
	ScanType    string
	ElapsedTime time.Duration
}

type Scanner interface {
	// ScanAll starts a scan of all libraries. This is a blocking operation.
	ScanAll(ctx context.Context, fullScan bool) (warnings []string, err error)
	// ScanFolders scans specific library/folder pairs, recursing into subdirectories.
	// If targets is nil, it scans all libraries. This is a blocking operation.
	ScanFolders(ctx context.Context, fullScan bool, targets []ScanTarget) (warnings []string, err error)
	Status(context.Context) (*ScannerStatus, error)
}

// ParseTargets parses scan targets strings into ScanTarget structs.
// Example: []string{"1:Music/Rock", "2:Classical"}
func ParseTargets(libFolders []string) ([]ScanTarget, error) {
	targets := make([]ScanTarget, 0, len(libFolders))

	for _, part := range libFolders {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Split by the first colon
		colonIdx := strings.Index(part, ":")
		if colonIdx == -1 {
			return nil, fmt.Errorf("invalid target format: %q (expected libraryID:folderPath)", part)
		}

		libIDStr := part[:colonIdx]
		folderPath := part[colonIdx+1:]

		libID, err := strconv.Atoi(libIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid library ID %q: %w", libIDStr, err)
		}
		if libID <= 0 {
			return nil, fmt.Errorf("invalid library ID %q", libIDStr)
		}

		targets = append(targets, ScanTarget{
			LibraryID:  libID,
			FolderPath: folderPath,
		})
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("no valid targets found")
	}

	return targets, nil
}
