package conf

// sqliteOptions configures SQLite database behavior
type sqliteOptions struct {
	// JournalMode sets the SQLite journal mode (WAL, DELETE, etc)
	// Default: WAL - provides better concurrency but may not work on network filesystems
	JournalMode string `json:",omitzero"`

	// BusyTimeout sets how long SQLite should wait for locks to clear (milliseconds)
	// Default: 5000 - waits up to 5 seconds before returning "database is locked"
	BusyTimeout int `json:",omitzero"`

	// SyncMode controls how aggressively SQLite writes to disk
	// Default: NORMAL - good balance of safety and performance
	SyncMode string `json:",omitzero"`

	// MaxConnections limits concurrent database connections
	// Default: 0 (uses max(4, runtime.NumCPU()))
	MaxConnections int `json:",omitzero"`
}
