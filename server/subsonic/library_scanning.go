package subsonic

import (
	"net/http"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils"
)

type LibraryScanningController struct {
	scanner scanner.Scanner
}

func NewLibraryScanningController(scanner scanner.Scanner) *LibraryScanningController {
	return &LibraryScanningController{scanner: scanner}
}

func (c *LibraryScanningController) GetScanStatus(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	// TODO handle multiple mediafolders
	ctx := r.Context()
	mediaFolder := conf.Server.MusicFolder
	status, err := c.scanner.Status(mediaFolder)
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
	}
	return response, nil
}

func (c *LibraryScanningController) StartScan(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	loggedUser, ok := request.UserFrom(ctx)
	if !ok {
		return nil, newError(responses.ErrorGeneric, "Internal error")
	}

	if !loggedUser.IsAdmin {
		return nil, newError(responses.ErrorAuthorizationFail)
	}

	fullScan := utils.ParamBool(r, "fullScan", false)

	go func() {
		start := time.Now()
		log.Info(ctx, "Triggering manual scan", "fullScan", fullScan, "user", loggedUser.UserName)
		err := c.scanner.RescanAll(ctx, fullScan)
		if err != nil {
			log.Error(ctx, "Error scanning", err)
			return
		}
		log.Info(ctx, "Manual scan complete", "user", loggedUser.UserName, "elapsed", time.Since(start).Round(100*time.Millisecond))
	}()

	return c.GetScanStatus(w, r)
}
