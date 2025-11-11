package subsonic

import (
	"fmt"
	"net/http"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/scanner"
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

	// Parse optional path parameters for selective scanning
	var targets []model.ScanTarget
	if pathParams, err := p.Strings("path"); err == nil && len(pathParams) > 0 {
		targets, err = scanner.ParseTargets(pathParams)
		if err != nil {
			return nil, newError(responses.ErrorGeneric, fmt.Sprintf("Invalid path parameter: %v", err))
		}
	}

	go func() {
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

	return api.GetScanStatus(r)
}
