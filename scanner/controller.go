package scanner

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/metrics"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/events"
	. "github.com/navidrome/navidrome/utils/gg"
	"github.com/navidrome/navidrome/utils/pl"
	"golang.org/x/time/rate"
)

var (
	ErrAlreadyScanning = errors.New("already scanning")
)

type Scanner interface {
	// ScanAll starts a full scan of the music library. This is a blocking operation.
	ScanAll(ctx context.Context, fullScan bool) (warnings []string, err error)
	Status(context.Context) (*StatusInfo, error)
}

type StatusInfo struct {
	Scanning    bool
	LastScan    time.Time
	Count       uint32
	FolderCount uint32
	LastError   string
	ScanType    string
	ElapsedTime time.Duration
}

func New(rootCtx context.Context, ds model.DataStore, cw artwork.CacheWarmer, broker events.Broker,
	pls core.Playlists, m metrics.Metrics) Scanner {
	c := &controller{
		rootCtx: rootCtx,
		ds:      ds,
		cw:      cw,
		broker:  broker,
		pls:     pls,
		metrics: m,
	}
	if !conf.Server.DevExternalScanner {
		c.limiter = P(rate.Sometimes{Interval: conf.Server.DevActivityPanelUpdateRate})
	}
	return c
}

func (s *controller) getScanner() scanner {
	if conf.Server.DevExternalScanner {
		return &scannerExternal{}
	}
	return &scannerImpl{ds: s.ds, cw: s.cw, pls: s.pls}
}

// CallScan starts an in-process scan of the music library.
// This is meant to be called from the command line (see cmd/scan.go).
func CallScan(ctx context.Context, ds model.DataStore, pls core.Playlists, fullScan bool) (<-chan *ProgressInfo, error) {
	release, err := lockScan(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	ctx = auth.WithAdminUser(ctx, ds)
	progress := make(chan *ProgressInfo, 100)
	go func() {
		defer close(progress)
		scanner := &scannerImpl{ds: ds, cw: artwork.NoopCacheWarmer(), pls: pls}
		scanner.scanAll(ctx, fullScan, progress)
	}()
	return progress, nil
}

func IsScanning() bool {
	return running.Load()
}

type ProgressInfo struct {
	LibID           int
	FileCount       uint32
	Path            string
	Phase           string
	ChangesDetected bool
	Warning         string
	Error           string
	ForceUpdate     bool
}

type scanner interface {
	scanAll(ctx context.Context, fullScan bool, progress chan<- *ProgressInfo)
}

type controller struct {
	rootCtx         context.Context
	ds              model.DataStore
	cw              artwork.CacheWarmer
	broker          events.Broker
	metrics         metrics.Metrics
	pls             core.Playlists
	limiter         *rate.Sometimes
	count           atomic.Uint32
	folderCount     atomic.Uint32
	changesDetected bool
}

// getLastScanTime returns the most recent scan time across all libraries
func (s *controller) getLastScanTime(ctx context.Context) (time.Time, error) {
	libs, err := s.ds.Library(ctx).GetAll(model.QueryOptions{
		Sort:  "last_scan_at",
		Order: "desc",
		Max:   1,
	})
	if err != nil {
		return time.Time{}, fmt.Errorf("getting libraries: %w", err)
	}

	if len(libs) == 0 {
		return time.Time{}, nil
	}

	return libs[0].LastScanAt, nil
}

// getScanInfo retrieves scan status from the database
func (s *controller) getScanInfo(ctx context.Context) (scanType string, elapsed time.Duration, lastErr string) {
	lastErr, _ = s.ds.Property(ctx).DefaultGet(consts.LastScanErrorKey, "")
	scanType, _ = s.ds.Property(ctx).DefaultGet(consts.LastScanTypeKey, "")
	startTimeStr, _ := s.ds.Property(ctx).DefaultGet(consts.LastScanStartTimeKey, "")

	if startTimeStr != "" {
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err == nil {
			if running.Load() {
				elapsed = time.Since(startTime)
			} else {
				// If scan is not running, calculate elapsed time using the most recent scan time
				lastScanTime, err := s.getLastScanTime(ctx)
				if err == nil && !lastScanTime.IsZero() {
					elapsed = lastScanTime.Sub(startTime)
				}
			}
		}
	}

	return scanType, elapsed, lastErr
}

func (s *controller) Status(ctx context.Context) (*StatusInfo, error) {
	lastScanTime, err := s.getLastScanTime(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting last scan time: %w", err)
	}

	scanType, elapsed, lastErr := s.getScanInfo(ctx)

	if running.Load() {
		status := &StatusInfo{
			Scanning:    true,
			LastScan:    lastScanTime,
			Count:       s.count.Load(),
			FolderCount: s.folderCount.Load(),
			LastError:   lastErr,
			ScanType:    scanType,
			ElapsedTime: elapsed,
		}
		return status, nil
	}

	count, folderCount, err := s.getCounters(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting library stats: %w", err)
	}
	return &StatusInfo{
		Scanning:    false,
		LastScan:    lastScanTime,
		Count:       uint32(count),
		FolderCount: uint32(folderCount),
		LastError:   lastErr,
		ScanType:    scanType,
		ElapsedTime: elapsed,
	}, nil
}

func (s *controller) getCounters(ctx context.Context) (int64, int64, error) {
	libs, err := s.ds.Library(ctx).GetAll()
	if err != nil {
		return 0, 0, fmt.Errorf("library count: %w", err)
	}
	var count, folderCount int64
	for _, l := range libs {
		count += int64(l.TotalSongs)
		folderCount += int64(l.TotalFolders)
	}
	return count, folderCount, nil
}

func (s *controller) ScanAll(requestCtx context.Context, fullScan bool) ([]string, error) {
	release, err := lockScan(requestCtx)
	if err != nil {
		return nil, err
	}
	defer release()

	// Prepare the context for the scan
	ctx := request.AddValues(s.rootCtx, requestCtx)
	ctx = auth.WithAdminUser(ctx, s.ds)

	// Send the initial scan status event
	s.sendMessage(ctx, &events.ScanStatus{Scanning: true, Count: 0, FolderCount: 0})
	progress := make(chan *ProgressInfo, 100)
	go func() {
		defer close(progress)
		scanner := s.getScanner()
		scanner.scanAll(ctx, fullScan, progress)
	}()

	// Wait for the scan to finish, sending progress events to all connected clients
	scanWarnings, scanError := s.trackProgress(ctx, progress)
	for _, w := range scanWarnings {
		log.Warn(ctx, fmt.Sprintf("Scan warning: %s", w))
	}
	// If changes were detected, send a refresh event to all clients
	if s.changesDetected {
		log.Debug(ctx, "Library changes imported. Sending refresh event")
		s.broker.SendBroadcastMessage(ctx, &events.RefreshResource{})
	}
	// Send the final scan status event, with totals
	if count, folderCount, err := s.getCounters(ctx); err != nil {
		s.metrics.WriteAfterScanMetrics(ctx, false)
		return scanWarnings, err
	} else {
		scanType, elapsed, lastErr := s.getScanInfo(ctx)
		s.metrics.WriteAfterScanMetrics(ctx, true)
		s.sendMessage(ctx, &events.ScanStatus{
			Scanning:    false,
			Count:       count,
			FolderCount: folderCount,
			Error:       lastErr,
			ScanType:    scanType,
			ElapsedTime: elapsed,
		})
	}
	return scanWarnings, scanError
}

// This is a global variable that is used to prevent multiple scans from running at the same time.
// "There can be only one" - https://youtu.be/sqcLjcSloXs?si=VlsjEOjTJZ68zIyg
var running atomic.Bool

func lockScan(ctx context.Context) (func(), error) {
	if !running.CompareAndSwap(false, true) {
		log.Debug(ctx, "Scanner already running, ignoring request")
		return func() {}, ErrAlreadyScanning
	}
	return func() {
		running.Store(false)
	}, nil
}

func (s *controller) trackProgress(ctx context.Context, progress <-chan *ProgressInfo) ([]string, error) {
	s.count.Store(0)
	s.folderCount.Store(0)
	s.changesDetected = false

	var warnings []string
	var errs []error
	for p := range pl.ReadOrDone(ctx, progress) {
		if p.Error != "" {
			errs = append(errs, errors.New(p.Error))
			continue
		}
		if p.Warning != "" {
			warnings = append(warnings, p.Warning)
			continue
		}
		if p.ChangesDetected {
			s.changesDetected = true
			continue
		}
		s.count.Add(p.FileCount)
		if p.FileCount > 0 {
			s.folderCount.Add(1)
		}

		scanType, elapsed, lastErr := s.getScanInfo(ctx)
		status := &events.ScanStatus{
			Scanning:    true,
			Count:       int64(s.count.Load()),
			FolderCount: int64(s.folderCount.Load()),
			Error:       lastErr,
			ScanType:    scanType,
			ElapsedTime: elapsed,
		}
		if s.limiter != nil && !p.ForceUpdate {
			s.limiter.Do(func() { s.sendMessage(ctx, status) })
		} else {
			s.sendMessage(ctx, status)
		}
	}
	return warnings, errors.Join(errs...)
}

func (s *controller) sendMessage(ctx context.Context, status *events.ScanStatus) {
	s.broker.SendBroadcastMessage(ctx, status)
}
