package rustworker

import (
	"context"
	"errors"
	"os"
	"testing"
)

func TestSkipPreflightInTests(t *testing.T) {
	if !SkipPreflightInTests(ErrSkippedInTests) {
		t.Fatal("expected ErrSkippedInTests to skip preflight")
	}
	if SkipPreflightInTests(errors.New("other")) {
		t.Fatal("unexpected skip for unrelated error")
	}
}

func TestGRPCWorkerCheckSkipsInTests(t *testing.T) {
	if os.Getenv("ND_GRPCWORKERINTESTS") != "" {
		t.Skip("ND_GRPCWORKERINTESTS enables real StartGRPC")
	}
	check := GRPCWorkerCheck{
		Name: "test",
		Path: "/bin/true",
		Health: func(context.Context, *GRPCProcess) error {
			return nil
		},
	}
	// StartGRPC is skipped in tests; health-only checks with a valid path should pass.
	if err := check.Run(context.Background()); err != nil {
		t.Fatalf("Run() in tests: %v", err)
	}
}

func TestPreflightGRPCStrictKeepEmpty(t *testing.T) {
	kept, err := PreflightGRPCStrictKeep(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(kept) != 0 {
		t.Fatalf("expected empty map, got %d", len(kept))
	}
}

func TestPreflightGRPCStrictKeepSkipsInTests(t *testing.T) {
	if os.Getenv("ND_GRPCWORKERINTESTS") != "" {
		t.Skip("ND_GRPCWORKERINTESTS enables real StartGRPC")
	}
	kept, err := PreflightGRPCStrictKeep(context.Background(), []GRPCWorkerCheck{{
		Name: "test",
		Path: "/bin/true",
		Health: func(context.Context, *GRPCProcess) error {
			return nil
		},
	}})
	if err != nil {
		t.Fatalf("StrictKeep in tests: %v", err)
	}
	if len(kept) != 0 {
		t.Fatalf("StartGRPC skipped in tests; expected no kept procs, got %d", len(kept))
	}
}
