package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/navidrome/navidrome/core/integration/gen"
	"github.com/navidrome/navidrome/core/rustworker"
)

type signVector struct {
	Secret   string            `json:"secret"`
	Params   map[string]string `json:"params"`
	Expected string            `json:"expected"`
}

func loadSignVector(t *testing.T) signVector {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "../../rust/integration/testdata/sign_vector.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var vector signVector
	if err := json.Unmarshal(raw, &vector); err != nil {
		t.Fatal(err)
	}
	return vector
}

func loadBlockedSSRFIPs(t *testing.T) []string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "../../rust/integration/testdata/ssrf_blocked_ips.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var ips []string
	if err := json.Unmarshal(raw, &ips); err != nil {
		t.Fatal(err)
	}
	return ips
}

func TestSignAudioscrobblerSharedVector(t *testing.T) {
	vector := loadSignVector(t)
	sig := signAudioscrobbler(vector.Params, vector.Secret)
	if sig != vector.Expected {
		t.Fatalf("sig = %s want %s", sig, vector.Expected)
	}
}

func TestIntegrationWorkerGRPC(t *testing.T) {
	binary := os.Getenv("ND_INTEGRATIONWORKERPATH")
	if binary == "" {
		t.Skip("ND_INTEGRATIONWORKERPATH not set")
	}
	if _, err := os.Stat(binary); err != nil {
		t.Skipf("integration worker binary missing: %v", err)
	}
	t.Setenv("ND_GRPCWORKERINTESTS", "1")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	proc, err := rustworker.StartGRPC(ctx, binary, rustworker.DefaultListenAddr("integration-test"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer proc.Close()

	client := gen.NewOutboundClient(proc.Conn)
	health, err := client.Health(ctx, &gen.HealthRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if !health.GetOk() {
		t.Fatal("health not ok")
	}

	sign, err := client.Sign(ctx, &gen.SignRequest{
		Params: loadSignVector(t).Params,
		Secret: loadSignVector(t).Secret,
	})
	if err != nil {
		t.Fatal(err)
	}
	if sign.GetApiSig() != loadSignVector(t).Expected {
		t.Fatalf("sign = %s want %s", sign.GetApiSig(), loadSignVector(t).Expected)
	}
}
