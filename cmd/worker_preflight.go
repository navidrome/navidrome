package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/apikeyworker"
	apikeysgen "github.com/navidrome/navidrome/core/apikeyworker/gen"
	"github.com/navidrome/navidrome/core/integration"
	"github.com/navidrome/navidrome/core/integration/gen"
	"github.com/navidrome/navidrome/core/metadataworker"
	metadatagen "github.com/navidrome/navidrome/core/metadataworker/gen"
	"github.com/navidrome/navidrome/core/rustworker"
	"github.com/navidrome/navidrome/core/scannerworker"
	scannergen "github.com/navidrome/navidrome/core/scannerworker/gen"
	"github.com/navidrome/navidrome/core/searchworker"
	searchgen "github.com/navidrome/navidrome/core/searchworker/gen"
)

var errWorkerHealthNotOK = errors.New("worker health check returned not ok")

func preflightRustWorkers(ctx context.Context) error {
	checks := make([]rustworker.GRPCWorkerCheck, 0, 4)
	if path, err := metadataworker.Resolve(); err == nil {
		checks = append(checks, rustworker.GRPCWorkerCheck{
			Name:     "metadata",
			Path:     path,
			MinBytes: rustworker.MinMetadataBytes,
			Health:   metadataGRPCHealth,
		})
	}
	if path, err := scannerworker.Resolve(); err == nil {
		checks = append(checks, rustworker.GRPCWorkerCheck{
			Name:     "scanner",
			Path:     path,
			MinBytes: rustworker.MinScannerBytes,
			Health:   scannerGRPCHealth,
		})
	}
	if path, err := searchworker.Resolve(); err == nil {
		var extraEnv []string
		if indexPath := searchworker.IndexPath(); indexPath != "" {
			extraEnv = []string{"NAVIDROME_SEARCH_INDEX_PATH=" + indexPath}
		}
		checks = append(checks, rustworker.GRPCWorkerCheck{
			Name:     "search",
			Path:     path,
			MinBytes: rustworker.MinSearchBytes,
			ExtraEnv: extraEnv,
			Health:   searchGRPCHealth,
		})
	}
	if path, err := apikeyworker.Resolve(); err == nil {
		checks = append(checks, rustworker.GRPCWorkerCheck{
			Name:     "apikeys",
			Path:     path,
			MinBytes: rustworker.MinApikeysBytes,
			Health:   apikeysGRPCHealth,
		})
	}
	if len(checks) > 0 {
		kept, err := rustworker.PreflightGRPCStrictKeep(ctx, checks)
		if err != nil {
			return fmt.Errorf("rust gRPC worker preflight: %w", err)
		}
		adoptPreflightWorkers(kept)
	}

	if !conf.Server.Integration.Enabled {
		return nil
	}
	path, err := integration.Resolve()
	if err != nil {
		return fmt.Errorf("integration worker binary: %w", err)
	}
	// Integration uses its own gateway lifecycle, not ManagedGRPC — keep
	// spawn-and-close preflight (no adopt).
	return rustworker.PreflightGRPCStrict(ctx, []rustworker.GRPCWorkerCheck{{
		Name:     "integration",
		Path:     path,
		MinBytes: rustworker.MinIntegrationBytes,
		Health:   integrationGRPCHealth,
	}})
}

func adoptPreflightWorkers(kept map[string]*rustworker.GRPCProcess) {
	for name, proc := range kept {
		adopted := false
		switch name {
		case "metadata":
			adopted = metadataworker.AdoptGRPC(proc)
		case "scanner":
			adopted = scannerworker.AdoptGRPC(proc)
		case "search":
			adopted = searchworker.AdoptGRPC(proc)
		case "apikeys":
			adopted = apikeyworker.AdoptGRPC(proc)
		}
		if !adopted {
			proc.Close()
		}
	}
}

func metadataGRPCHealth(ctx context.Context, proc *rustworker.GRPCProcess) error {
	client := metadatagen.NewMetadataClient(proc.Conn)
	resp, err := client.Health(ctx, &metadatagen.HealthRequest{})
	if err != nil {
		return err
	}
	if !resp.GetOk() {
		return errWorkerHealthNotOK
	}
	return nil
}

func scannerGRPCHealth(ctx context.Context, proc *rustworker.GRPCProcess) error {
	client := scannergen.NewFolderHashClient(proc.Conn)
	resp, err := client.Health(ctx, &scannergen.HealthRequest{})
	if err != nil {
		return err
	}
	if !resp.GetOk() {
		return errWorkerHealthNotOK
	}
	return nil
}

func searchGRPCHealth(ctx context.Context, proc *rustworker.GRPCProcess) error {
	client := searchgen.NewSearchClient(proc.Conn)
	resp, err := client.Health(ctx, &searchgen.HealthRequest{})
	if err != nil {
		return err
	}
	if !resp.GetOk() {
		return errWorkerHealthNotOK
	}
	return nil
}

func apikeysGRPCHealth(ctx context.Context, proc *rustworker.GRPCProcess) error {
	client := apikeysgen.NewApiKeysClient(proc.Conn)
	resp, err := client.Health(ctx, &apikeysgen.HealthRequest{})
	if err != nil {
		return err
	}
	if !resp.GetOk() {
		return errWorkerHealthNotOK
	}
	return nil
}

func integrationGRPCHealth(ctx context.Context, proc *rustworker.GRPCProcess) error {
	client := gen.NewOutboundClient(proc.Conn)
	resp, err := client.Health(ctx, &gen.HealthRequest{})
	if err != nil {
		return err
	}
	if !resp.GetOk() {
		return errIntegrationHealthNotOK
	}
	return nil
}

// warmRustWorkers starts managed gRPC companions after preflight so the first
// request avoids a cold spawn. Workers already adopted from preflight make
// Warm a cheap Conn hit; only unresolved binaries pay a background spawn.
func warmRustWorkers() {
	metadataworker.WarmGRPC()
	scannerworker.WarmGRPC()
	searchworker.WarmGRPC()
	apikeyworker.WarmGRPC()
}
