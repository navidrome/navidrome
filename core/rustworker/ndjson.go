package rustworker

import (
	"github.com/navidrome/navidrome/log"
)

// LogGRPCUnavailable records that a Rust gRPC worker could not be started.
func LogGRPCUnavailable(name string, err error) {
	log.Error("Rust "+name+" gRPC worker unavailable", err)
}
