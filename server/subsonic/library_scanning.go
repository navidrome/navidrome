package subsonic

import (
	"net/http"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model/request"
	"github.com/deluan/navidrome/scanner"
	"github.com/deluan/navidrome/server/subsonic/responses"
	"github.com/deluan/navidrome/utils"
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
	loggedUser, ok := request.UserFrom(r.Context())
	if !ok {
		return nil, newError(responses.ErrorGeneric, "Internal error")
	}

	if !loggedUser.IsAdmin {
		return nil, newError(responses.ErrorAuthorizationFail)
	}

	fullScan := utils.ParamBool(r, "fullScan", false)

	go func() {
		err := c.scanner.RescanAll(fullScan)
		if err != nil {
			log.Error(r.Context(), err)
		}
	}()

	return c.GetScanStatus(w, r)
}
