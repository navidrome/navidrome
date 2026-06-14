package model

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ErrAlreadyScanning is returned when a scan is requested while another scan is
// already in progress. It is defined here (not in the scanner package) so that
// packages depending only on the model interfaces can match it with errors.Is
// without importing the scanner package.
var ErrAlreadyScanning = errors.New("already scanning")

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
		before, after, ok := strings.Cut(part, ":")
		if !ok {
			return nil, fmt.Errorf("invalid target format: %q (expected libraryID:folderPath)", part)
		}

		libIDStr := before
		folderPath := after

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
