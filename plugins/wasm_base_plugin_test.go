package plugins

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type nilInstance struct{}

var _ = Describe("wasmBasePlugin", func() {
	var ctx = context.Background()

	It("should load instance using loadFunc", func() {
		called := false
		plugin := &wasmBasePlugin[*nilInstance, any]{
			wasmPath:   "",
			id:         "test",
			capability: "test",
			loadFunc: func(ctx context.Context, _ any, path string) (*nilInstance, error) {
				called = true
				return &nilInstance{}, nil
			},
		}
		inst, done, err := plugin.getInstance(ctx, "test")
		defer done()
		Expect(err).To(BeNil())
		Expect(inst).ToNot(BeNil())
		Expect(called).To(BeTrue())
	})
})
