package subsonic

import (
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
)

func (api *Router) GetScanStatus(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	status, err := api.scanner.Status(ctx)
	if err != nil {
		log.Error(ctx, "Error retrieving Scanner status", err)
		return nil, newError(responses.ErrorGeneric, "Internal Error")
	}
	response := newResponse()
	response.ScanStatus = &responses.ScanStatus{
		Scanning:    status.Scanning,
		Count:       int64(status.Count),
		FolderCount: int64(status.FolderCount),
		LastScan:    &status.LastScan,
		Error:       status.LastError,
		ScanType:    status.ScanType,
		ElapsedTime: int64(status.ElapsedTime),
	}
	return response, nil
}

func (api *Router) StartScan(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	loggedUser, ok := request.UserFrom(ctx)
	if !ok {
		return nil, newError(responses.ErrorGeneric, "Internal error")
	}

	if !loggedUser.IsAdmin {
		return nil, newError(responses.ErrorAuthorizationFail)
	}

	p := req.Params(r)
	fullScan := p.BoolOr("fullScan", false)

	// Parse optional target parameters for selective scanning
	var targets []model.ScanTarget
	if targetParams, err := p.Strings("target"); err == nil && len(targetParams) > 0 {
		targets, err = model.ParseTargets(targetParams)
		if err != nil {
			return nil, newError(responses.ErrorGeneric, fmt.Sprintf("Invalid target parameter: %v", err))
		}

		// Validate all libraries in targets exist and user has access to them
		userLibraries, err := api.ds.User(ctx).GetUserLibraries(loggedUser.ID)
		if err != nil {
			return nil, newError(responses.ErrorGeneric, "Internal error")
		}

		// Check each target library
		for _, target := range targets {
			if !slices.ContainsFunc(userLibraries, func(lib model.Library) bool { return lib.ID == target.LibraryID }) {
				return nil, newError(responses.ErrorDataNotFound, fmt.Sprintf("Library with ID %d not found", target.LibraryID))
			}
		}

		// Special case: if single library with empty path and it's the only library in DB, call ScanAll
		if len(targets) == 1 && targets[0].FolderPath == "" {
			allLibs, err := api.ds.Library(ctx).GetAll()
			if err != nil {
				return nil, newError(responses.ErrorGeneric, "Internal error")
			}
			if len(allLibs) == 1 {
				targets = nil // This will trigger ScanAll below
			}
		}
	}

	fastScanCompleted := make(chan struct{})
	go func() {
		defer close(fastScanCompleted)
		start := time.Now()
		var err error

		if len(targets) > 0 {
			log.Info(ctx, "Triggering on-demand scan", "fullScan", fullScan, "targets", len(targets), "user", loggedUser.UserName)
			_, err = api.scanner.ScanFolders(ctx, fullScan, targets)
		} else {
			log.Info(ctx, "Triggering on-demand scan", "fullScan", fullScan, "user", loggedUser.UserName)
			_, err = api.scanner.ScanAll(ctx, fullScan)
		}

		if err != nil {
			log.Error(ctx, "Error scanning", err)
			return
		}
		log.Info(ctx, "On-demand scan complete", "user", loggedUser.UserName, "elapsed", time.Since(start))
	}()

	// Wait briefly for the scanner to start and update its status, so the response
	// reflects the current scan (not stale data from a previous scan).
	const (
		pollInterval = 50 * time.Millisecond
		pollTimeout  = 3 * time.Second
	)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	timer := time.NewTimer(pollTimeout)
	defer timer.Stop()

loop:
	for {
		status, err := api.scanner.Status(ctx)
		if err == nil && status.Scanning {
			break
		}
		select {
		case <-fastScanCompleted:
			log.Info(ctx, "Fast scan completed", "user", loggedUser.UserName)
			break loop
		case <-timer.C:
			log.Warn(ctx, "Timed out waiting for scanner to start; response may be stale")
			break loop
		case <-ctx.Done():
			return nil, newError(responses.ErrorGeneric, "Request cancelled while waiting for scanner to start")
		case <-ticker.C:
		}
	}

	return api.GetScanStatus(r)
}
