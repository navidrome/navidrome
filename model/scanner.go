package model

import (
	"fmt"
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
