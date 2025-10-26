package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	sqliteLockWaitDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "sqlite_lock_wait_duration_seconds",
			Help:    "Time spent waiting for SQLite locks to be released",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"operation"},
	)

	sqliteLockErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sqlite_lock_errors_total",
			Help: "Number of SQLite lock-related errors",
		},
		[]string{"operation", "type"},
	)

	sqliteRetries = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sqlite_operation_retries_total",
			Help: "Number of retried SQLite operations",
		},
		[]string{"operation"},
	)
)

func init() {
	prometheus.MustRegister(sqliteLockWaitDuration)
	prometheus.MustRegister(sqliteLockErrors)
	prometheus.MustRegister(sqliteRetries)
}

// ObserveSQLiteLockWait records the duration spent waiting for a lock
func ObserveSQLiteLockWait(operation string, duration float64) {
	sqliteLockWaitDuration.WithLabelValues(operation).Observe(duration)
}

// IncrementSQLiteLockError increments the counter for lock-related errors
func IncrementSQLiteLockError(operation, errType string) {
	sqliteLockErrors.WithLabelValues(operation, errType).Inc()
}

// IncrementSQLiteRetry increments the counter for operation retries
func IncrementSQLiteRetry(operation string) {
	sqliteRetries.WithLabelValues(operation).Inc()
}
