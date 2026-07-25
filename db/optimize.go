package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
)

var analyzeMux sync.Mutex

// Optimize refreshes the query-planner statistics with a full ANALYZE. PRAGMA optimize is avoided
// because its limited analysis misestimates Navidrome's low-cardinality indexes.
func Optimize(ctx context.Context) error {
	analyzeMux.Lock()
	defer analyzeMux.Unlock()
	start := time.Now()
	if err := optimizeAt(ctx, Db(), start); err != nil {
		return err
	}
	log.Info(ctx, "DB analysis complete", "elapsed", time.Since(start))
	return nil
}

// OptimizeIfNeeded refreshes statistics when they are stale or a database-changing operation
// marked them for refresh.
func OptimizeIfNeeded(ctx context.Context) (bool, error) {
	analyzeMux.Lock()
	defer analyzeMux.Unlock()
	start := time.Now()
	ran, err := optimizeIfNeeded(ctx, Db(), start)
	if err != nil || !ran {
		return ran, err
	}
	log.Info(ctx, "DB analysis complete", "elapsed", time.Since(start))
	return true, nil
}

func optimizeIfNeeded(ctx context.Context, db *sql.DB, now time.Time) (bool, error) {
	due, err := optimizeDue(ctx, db, now)
	if err != nil || !due {
		return false, err
	}
	return true, optimizeAt(ctx, db, now)
}

func optimizeDue(ctx context.Context, db *sql.DB, now time.Time) (bool, error) {
	backingOff, err := analyzeRetryBackoffActive(ctx, db, now)
	if err != nil || backingOff {
		return false, err
	}

	pending, found, err := getProperty(ctx, db, consts.DBAnalyzePendingKey)
	if err != nil {
		return false, err
	}
	if found && pending == "1" {
		return true, nil
	}

	value, found, err := getProperty(ctx, db, consts.LastDBAnalyzeAtKey)
	if err != nil {
		return false, err
	}
	if !found {
		return true, nil
	}

	lastAnalyze, valid := parseAnalyzeTime(value)
	if !valid || lastAnalyze.After(now) {
		return true, nil
	}
	return now.Sub(lastAnalyze) >= consts.DBAnalyzeMaxAge, nil
}

func parseAnalyzeTime(value string) (time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	return parsed, err == nil
}

func analyzeRetryBackoffActive(ctx context.Context, db *sql.DB, now time.Time) (bool, error) {
	value, found, err := getProperty(ctx, db, consts.DBAnalyzeFailureCountKey)
	if err != nil || !found {
		return false, err
	}
	failures, _ := strconv.Atoi(value)
	if failures < 1 {
		return false, nil
	}

	value, found, err = getProperty(ctx, db, consts.LastDBAnalyzeAttemptAtKey)
	if err != nil || !found {
		return false, err
	}
	lastAttempt, valid := parseAnalyzeTime(value)
	if !valid || lastAttempt.After(now) {
		return false, nil
	}
	return now.Sub(lastAttempt) < analyzeRetryDelay(failures), nil
}

func analyzeRetryDelay(failures int) time.Duration {
	switch failures {
	case 1:
		return 30 * time.Minute
	case 2:
		return time.Hour
	case 3:
		return 2 * time.Hour
	default:
		return 24 * time.Hour
	}
}

// MarkOptimizePending requests a statistics refresh on the next scheduled maintenance check.
func MarkOptimizePending(ctx context.Context) error {
	analyzeMux.Lock()
	defer analyzeMux.Unlock()
	return markOptimizePending(ctx, Db())
}

func markOptimizePending(ctx context.Context, db *sql.DB) error {
	return putProperty(ctx, db, consts.DBAnalyzePendingKey, "1")
}

func optimizeAt(ctx context.Context, db *sql.DB, now time.Time) error {
	if err := markOptimizePending(ctx, db); err != nil {
		return recordAnalyzeError(ctx, db, now, fmt.Errorf("marking ANALYZE pending: %w", err))
	}
	log.Debug(ctx, "Refreshing query planner statistics")
	_, err := db.ExecContext(ctx, "ANALYZE")
	if err != nil {
		return recordAnalyzeError(ctx, db, now, fmt.Errorf("running ANALYZE: %w", err))
	}
	if err = recordAnalyzeSuccess(ctx, db, now); err != nil {
		return recordAnalyzeError(ctx, db, now, err)
	}
	return nil
}

func recordAnalyzeSuccess(ctx context.Context, db *sql.DB, now time.Time) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("recording ANALYZE time: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err = putProperty(ctx, tx, consts.LastDBAnalyzeAtKey, now.UTC().Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("recording ANALYZE time: %w", err)
	}
	if err = putProperty(ctx, tx, consts.DBAnalyzePendingKey, "0"); err != nil {
		return fmt.Errorf("clearing pending ANALYZE: %w", err)
	}
	if err = putProperty(ctx, tx, consts.DBAnalyzeFailureCountKey, "0"); err != nil {
		return fmt.Errorf("clearing ANALYZE failure count: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("recording ANALYZE state: %w", err)
	}
	return nil
}

func recordAnalyzeError(ctx context.Context, db *sql.DB, now time.Time, analyzeErr error) error {
	if err := recordAnalyzeFailure(ctx, db, now); err != nil {
		return errors.Join(analyzeErr, fmt.Errorf("recording ANALYZE failure: %w", err))
	}
	return analyzeErr
}

func recordAnalyzeFailure(ctx context.Context, db *sql.DB, now time.Time) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	value, found, err := getProperty(ctx, tx, consts.DBAnalyzeFailureCountKey)
	if err != nil {
		return err
	}
	failures := 0
	if found {
		failures, _ = strconv.Atoi(value)
		failures = max(failures, 0)
	}
	if err = putProperty(ctx, tx, consts.DBAnalyzePendingKey, "1"); err != nil {
		return err
	}
	if err = putProperty(ctx, tx, consts.DBAnalyzeFailureCountKey, strconv.Itoa(failures+1)); err != nil {
		return err
	}
	if err = putProperty(ctx, tx, consts.LastDBAnalyzeAttemptAtKey, now.UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	return tx.Commit()
}

type sqlExecer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type sqlQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func putProperty(ctx context.Context, db sqlExecer, key, value string) error {
	_, err := db.ExecContext(ctx, `insert into property(id, value) values(?, ?)
		on conflict(id) do update set value=excluded.value`, key, value)
	return err
}

func getProperty(ctx context.Context, db sqlQueryer, key string) (string, bool, error) {
	var value string
	err := db.QueryRowContext(ctx, "select value from property where id=?", key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return value, err == nil, err
}
