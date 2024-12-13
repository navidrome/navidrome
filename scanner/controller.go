package scanner

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/auth"
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
	ScanAll(ctx context.Context, fullScan bool) error
	Status(context.Context) (*StatusInfo, error)
}

type StatusInfo struct {
	Scanning    bool
	LastScan    time.Time
	Count       uint32
	FolderCount uint32
}

func New(rootCtx context.Context, ds model.DataStore, cw artwork.CacheWarmer, broker events.Broker, pls core.Playlists) Scanner {
	c := &controller{
		rootCtx: rootCtx,
		ds:      ds,
		cw:      cw,
		broker:  broker,
		pls:     pls,
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

// Scan starts a full scan of the music library. This is meant to be called from the command line (see cmd/scan.go).
func Scan(ctx context.Context, ds model.DataStore, cw artwork.CacheWarmer, pls core.Playlists, fullScan bool) (<-chan *ProgressInfo, error) {
	release, err := lockScan(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	ctx = auth.WithAdminUser(ctx, ds)
	progress := make(chan *ProgressInfo, 100)
	go func() {
		defer close(progress)
		scanner := &scannerImpl{ds: ds, cw: cw, pls: pls}
		scanner.scanAll(ctx, fullScan, progress)
	}()
	return progress, nil
}

type ProgressInfo struct {
	LibID           int
	FileCount       uint32
	FolderCount     uint32
	Path            string
	Phase           string
	ChangesDetected bool
	Err             string
}

type scanner interface {
	scanAll(ctx context.Context, fullScan bool, progress chan<- *ProgressInfo)
	// BFR: scanFolders(ctx context.Context, lib model.Lib, folders []string, progress chan<- *ScannerStatus)
}

type controller struct {
	rootCtx         context.Context
	ds              model.DataStore
	cw              artwork.CacheWarmer
	broker          events.Broker
	pls             core.Playlists
	limiter         *rate.Sometimes
	count           atomic.Uint32
	folderCount     atomic.Uint32
	changesDetected bool
}

func (s *controller) Status(ctx context.Context) (*StatusInfo, error) {
	lib, err := s.ds.Library(ctx).Get(1) //TODO Multi-library
	if err != nil {
		return nil, fmt.Errorf("getting library: %w", err)
	}
	if running.Load() {
		status := &StatusInfo{
			Scanning:    true,
			LastScan:    lib.LastScanAt,
			Count:       s.count.Load(),
			FolderCount: s.folderCount.Load(),
		}
		return status, nil
	}
	count, folderCount, err := s.getCounters(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting library stats: %w", err)
	}
	return &StatusInfo{
		Scanning:    false,
		LastScan:    lib.LastScanAt,
		Count:       uint32(count),
		FolderCount: uint32(folderCount),
	}, nil
}

func (s *controller) getCounters(ctx context.Context) (int64, int64, error) {
	count, err := s.ds.MediaFile(ctx).CountAll()
	if err != nil {
		return 0, 0, fmt.Errorf("media file count: %w", err)
	}
	folderCount, err := s.ds.Folder(ctx).CountAll()
	if err != nil {
		return 0, 0, fmt.Errorf("folder count: %w", err)
	}
	return count, folderCount, nil
}

func (s *controller) ScanAll(requestCtx context.Context, fullScan bool) error {
	release, err := lockScan(requestCtx)
	if err != nil {
		return err
	}
	defer release()

	// Prepare the context for the scan
	ctx := request.AddValues(s.rootCtx, requestCtx)
	ctx = events.BroadcastToAll(ctx)
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
	if err = s.trackProgress(ctx, progress); err != nil {
		return err
	}

	// If changes were detected, send a refresh event to all clients
	if s.changesDetected {
		log.Debug(ctx, "Library changes imported. Sending refresh event")
		s.broker.SendMessage(ctx, &events.RefreshResource{})
	}

	// Send the final scan status event, with totals
	count, folderCount, err := s.getCounters(ctx)
	if err != nil {
		return err
	}
	s.sendMessage(ctx, &events.ScanStatus{
		Scanning:    false,
		Count:       count,
		FolderCount: folderCount,
	})
	return nil
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

func (s *controller) trackProgress(ctx context.Context, progress <-chan *ProgressInfo) error {
	s.count.Store(0)
	s.folderCount.Store(0)
	s.changesDetected = false

	var errs []error
	for p := range pl.ReadOrDone(ctx, progress) {
		if p.Err != "" {
			errs = append(errs, errors.New(p.Err))
			continue
		}
		if p.ChangesDetected {
			s.changesDetected = true
			continue
		}
		s.count.Add(p.FileCount)
		s.folderCount.Add(1)
		status := &events.ScanStatus{
			Scanning:    true,
			Count:       int64(s.count.Load()),
			FolderCount: int64(s.folderCount.Load()),
		}
		if s.limiter != nil {
			s.limiter.Do(func() { s.sendMessage(ctx, status) })
		} else {
			s.sendMessage(ctx, status)
		}
	}
	return errors.Join(errs...)
}

func (s *controller) sendMessage(ctx context.Context, status *events.ScanStatus) {
	s.broker.SendMessage(ctx, status)
}
