package dlna

import (
	"context"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/events"
)

type DLNAServer struct {
	ds     model.DataStore
	broker events.Broker
}

func New(ds model.DataStore, broker events.Broker) *DLNAServer {
	s := &DLNAServer{ds: ds, broker: broker}
	return s
}

// Run starts the server with the given address, and if specified, with TLS enabled.
func (s *DLNAServer) Run(ctx context.Context, addr string, port int, tlsCert string, tlsKey string) error {
	return nil
}
