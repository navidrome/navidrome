package dto

// PublicSystemInfo is the unauthenticated handshake payload (GET /System/Info/Public).
type PublicSystemInfo struct {
	LocalAddress           string `json:"LocalAddress,omitempty"`
	ServerName             string `json:"ServerName"`
	Version                string `json:"Version"`
	ProductName            string `json:"ProductName"`
	OperatingSystem        string `json:"OperatingSystem,omitempty"`
	Id                     string `json:"Id"`
	StartupWizardCompleted bool   `json:"StartupWizardCompleted"`
}

// SystemInfo is the authenticated variant (GET /System/Info).
type SystemInfo struct {
	PublicSystemInfo
	HasPendingRestart      bool   `json:"HasPendingRestart"`
	IsShuttingDown         bool   `json:"IsShuttingDown"`
	SupportsLibraryMonitor bool   `json:"SupportsLibraryMonitor"`
	CachePath              string `json:"CachePath,omitempty"`
}
