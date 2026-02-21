//go:build go1.21

package dlna

import (
	"net/http"

	"github.com/anacrolix/dms/upnp"
)

const defaultProtocolInfo = "http-get:*:audio/mpeg:*,http-get:*:audio/mp3:*,http-get:*:audio/mp4:*,http-get:*:audio/x-ms-wma*,http-get:*:audio/wav:*,http-get:*:audio/L16:*,http-get:*image/jpeg:*,http-get:*image/png:*,http-get:*image/gif:*,http-get:*image/tiff:*"

type connectionManagerService struct {
	*DLNAServer
	upnp.Eventing
}

func (cms *connectionManagerService) Handle(action string, argsXML []byte, r *http.Request) (map[string]string, error) {
	switch action {
	case "GetProtocolInfo":
		return map[string]string{
			"Source": defaultProtocolInfo,
			"Sink":   "",
		}, nil
	default:
		return nil, upnp.InvalidActionError
	}
}
