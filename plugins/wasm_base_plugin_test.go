package plugins

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type nilInstance struct{}

var _ = Describe("wasmBasePlugin", func() {
	var ctx = context.Background()

	It("should return an error if the pool returns nil", func() {
		pool := NewWasmInstancePool[*nilInstance]("test-nil-pool", 1, time.Second, func(ctx context.Context) (*nilInstance, error) {
			return nil, fmt.Errorf("forced nil instance")
		})
		plugin := &wasmBasePlugin[*nilInstance, any]{
			pool:     pool,
			wasmPath: "",
			name:     "test-nil-pool",
			service:  "test",
		}
		plugin.poolOnce.Do(func() {}) // Don't init pool again
		inst, done, err := plugin.getInstance(ctx, "testMethod")
		defer done()
		Expect(inst).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to get instance"))
	})
})
