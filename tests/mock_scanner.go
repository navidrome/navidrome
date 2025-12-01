package tests

import (
	"context"
	"sync"

	"github.com/navidrome/navidrome/model"
)

// MockScanner implements scanner.Scanner for testing with proper synchronization
type MockScanner struct {
	mu               sync.Mutex
	scanAllCalls     []ScanAllCall
	scanFoldersCalls []ScanFoldersCall
	scanningStatus   bool
	statusResponse   *model.ScannerStatus
}

type ScanAllCall struct {
	FullScan bool
}

type ScanFoldersCall struct {
	FullScan bool
	Targets  []model.ScanTarget
}

func NewMockScanner() *MockScanner {
	return &MockScanner{
		scanAllCalls:     make([]ScanAllCall, 0),
		scanFoldersCalls: make([]ScanFoldersCall, 0),
	}
}

func (m *MockScanner) ScanAll(_ context.Context, fullScan bool) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.scanAllCalls = append(m.scanAllCalls, ScanAllCall{FullScan: fullScan})

	return nil, nil
}

func (m *MockScanner) ScanFolders(_ context.Context, fullScan bool, targets []model.ScanTarget) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Make a copy of targets to avoid race conditions
	targetsCopy := make([]model.ScanTarget, len(targets))
	copy(targetsCopy, targets)

	m.scanFoldersCalls = append(m.scanFoldersCalls, ScanFoldersCall{
		FullScan: fullScan,
		Targets:  targetsCopy,
	})

	return nil, nil
}

func (m *MockScanner) Status(_ context.Context) (*model.ScannerStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.statusResponse != nil {
		return m.statusResponse, nil
	}

	return &model.ScannerStatus{
		Scanning: m.scanningStatus,
	}, nil
}

func (m *MockScanner) GetScanAllCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.scanAllCalls)
}

func (m *MockScanner) GetScanAllCalls() []ScanAllCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Return a copy to avoid race conditions
	calls := make([]ScanAllCall, len(m.scanAllCalls))
	copy(calls, m.scanAllCalls)
	return calls
}

func (m *MockScanner) GetScanFoldersCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.scanFoldersCalls)
}

func (m *MockScanner) GetScanFoldersCalls() []ScanFoldersCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Return a copy to avoid race conditions
	calls := make([]ScanFoldersCall, len(m.scanFoldersCalls))
	copy(calls, m.scanFoldersCalls)
	return calls
}

func (m *MockScanner) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scanAllCalls = make([]ScanAllCall, 0)
	m.scanFoldersCalls = make([]ScanFoldersCall, 0)
}

func (m *MockScanner) SetScanning(scanning bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scanningStatus = scanning
}

func (m *MockScanner) SetStatusResponse(status *model.ScannerStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statusResponse = status
}
