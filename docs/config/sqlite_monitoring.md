# SQLite Monitoring

Navidrome provides several Prometheus metrics to monitor SQLite database performance and lock contention:

## Available Metrics

### Lock Wait Duration

- **Metric**: `sqlite_lock_wait_duration_seconds`
- **Type**: Histogram
- **Labels**: `operation`
- **Description**: Time spent waiting for SQLite locks to be released
- **Use Case**: Identify operations that are frequently blocked by locks

### Lock Errors

- **Metric**: `sqlite_lock_errors_total`
- **Type**: Counter
- **Labels**: `operation`, `type`
- **Description**: Number of SQLite lock-related errors
- **Types**:
  - `retryable`: Temporary lock errors that can be retried
  - `non_retryable`: Fatal errors that cannot be retried

### Operation Retries

- **Metric**: `sqlite_operation_retries_total`
- **Type**: Counter
- **Labels**: `operation`
- **Description**: Number of retried SQLite operations
- **Use Case**: Track which operations require frequent retries

## Grafana Dashboard

A pre-configured Grafana dashboard is available at `contrib/grafana/sqlite-dashboard.json`.
This dashboard provides visualizations for:

- Lock wait duration trends
- Lock error rates by operation
- Retry rates by operation

## Alerting Recommendations

Consider setting up alerts for:

1. High lock wait durations (> 1s)
2. Increasing error rates
3. Frequent retries on specific operations

Example Prometheus alert rules:

```yaml
groups:
  - name: SQLiteAlerts
    rules:
      - alert: SQLiteLongLockWaits
        expr: rate(sqlite_lock_wait_duration_seconds_sum[5m]) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          description: "SQLite operations are waiting long for locks"

      - alert: SQLiteHighErrorRate
        expr: rate(sqlite_lock_errors_total[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          description: "High rate of SQLite lock errors"
```

## Troubleshooting with Metrics

If you see:

1. High lock wait durations:

   - Consider adjusting `SQLite.BusyTimeout`
   - Review concurrent operations
   - Check if database is on network storage

2. Many retryable errors:

   - Increase `SQLite.BusyTimeout`
   - Consider reducing `SQLite.MaxConnections`
   - Enable WAL mode if not using network storage

3. High retry rates:
   - Review operations causing contention
   - Consider batching updates
   - Check for long-running transactions
