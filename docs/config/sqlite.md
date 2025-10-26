# SQLite Configuration Options

The following SQLite-specific configuration options are available under the `SQLite` section:

## Basic Options

### JournalMode

- **Default:** `"WAL"`
- **Options:** `"DELETE"`, `"TRUNCATE"`, `"PERSIST"`, `"MEMORY"`, `"WAL"`, `"OFF"`
- **Description:** Controls how SQLite manages its journal file. WAL (Write-Ahead Logging) mode generally provides better concurrency and performance but may not work on some network filesystems.

### BusyTimeout

- **Default:** `5000` (milliseconds)
- **Description:** How long SQLite should wait when the database is locked before returning a "database is locked" error. Higher values allow more concurrency but may impact responsiveness.

### SyncMode

- **Default:** `"NORMAL"`
- **Options:** `"OFF"`, `"NORMAL"`, `"FULL"`, `"EXTRA"`
- **Description:** Controls how aggressively SQLite writes data to disk. NORMAL provides a good balance between safety and performance.

### MaxConnections

- **Default:** `0` (uses max(4, number of CPU cores))
- **Description:** Maximum number of concurrent database connections. Lower this if you experience "database is locked" errors, especially on network filesystems.

## Example Configuration

```toml
[SQLite]
JournalMode = "WAL"      # Enable Write-Ahead Logging for better concurrency
BusyTimeout = 5000       # Wait up to 5 seconds for locks to clear
SyncMode = "NORMAL"      # Good balance of durability and performance
MaxConnections = 4       # Limit concurrent connections if needed
```

## Network Filesystem Considerations

If your database is on a network filesystem (NFS, CIFS, etc.):

1. Consider moving the database to local storage
2. If using network storage is required:
   - Set `JournalMode = "DELETE"`
   - Lower `MaxConnections` to reduce contention
   - Increase `BusyTimeout` for better reliability
