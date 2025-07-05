package plugins

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type nilInstance struct{}

var _ = Describe("baseCapability", func() {
	var ctx = context.Background()

	It("should load instance using loadFunc", func() {
		called := false
		plugin := &baseCapability[*nilInstance, any]{
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

var _ = Describe("checkErr", func() {
	Context("when resp is nil", func() {
		It("should return the original error unchanged", func() {
			var resp *testErrorResponse
			originalErr := errors.New("original error")

			result, err := checkErr(resp, originalErr)

			Expect(result).To(BeNil())
			Expect(err).To(Equal(originalErr))
		})

		It("should return nil error when both resp and err are nil", func() {
			var resp *testErrorResponse

			result, err := checkErr(resp, nil)

			Expect(result).To(BeNil())
			Expect(err).To(BeNil())
		})
	})

	Context("when resp is a typed nil that implements errorResponse", func() {
		It("should not panic and return original error", func() {
			var resp *testErrorResponse // typed nil
			originalErr := errors.New("original error")

			// This should not panic
			result, err := checkErr(resp, originalErr)

			Expect(result).To(BeNil())
			Expect(err).To(Equal(originalErr))
		})

		It("should handle typed nil with nil error gracefully", func() {
			var resp *testErrorResponse // typed nil

			// This should not panic
			result, err := checkErr(resp, nil)

			Expect(result).To(BeNil())
			Expect(err).To(BeNil())
		})
	})

	Context("when resp implements errorResponse with non-empty error", func() {
		It("should create new error when original error is nil", func() {
			resp := &testErrorResponse{errorMsg: "plugin error"}

			result, err := checkErr(resp, nil)

			Expect(result).To(Equal(resp))
			Expect(err).To(MatchError("plugin error"))
		})

		It("should wrap original error when both exist", func() {
			resp := &testErrorResponse{errorMsg: "plugin error"}
			originalErr := errors.New("original error")

			result, err := checkErr(resp, originalErr)

			Expect(result).To(Equal(resp))
			Expect(err).To(MatchError("plugin error: original error"))
		})
	})

	Context("when resp implements errorResponse with empty error", func() {
		It("should return original error unchanged", func() {
			resp := &testErrorResponse{errorMsg: ""}
			originalErr := errors.New("original error")

			result, err := checkErr(resp, originalErr)

			Expect(result).To(Equal(resp))
			Expect(err).To(Equal(originalErr))
		})

		It("should return nil error when both are empty/nil", func() {
			resp := &testErrorResponse{errorMsg: ""}

			result, err := checkErr(resp, nil)

			Expect(result).To(Equal(resp))
			Expect(err).To(BeNil())
		})
	})

	Context("when resp does not implement errorResponse", func() {
		It("should return original error unchanged", func() {
			resp := &testNonErrorResponse{data: "some data"}
			originalErr := errors.New("original error")

			result, err := checkErr(resp, originalErr)

			Expect(result).To(Equal(resp))
			Expect(err).To(Equal(originalErr))
		})

		It("should return nil error when original error is nil", func() {
			resp := &testNonErrorResponse{data: "some data"}

			result, err := checkErr(resp, nil)

			Expect(result).To(Equal(resp))
			Expect(err).To(BeNil())
		})
	})

	Context("when resp is a value type (not pointer)", func() {
		It("should handle value types that implement errorResponse", func() {
			resp := testValueErrorResponse{errorMsg: "value error"}
			originalErr := errors.New("original error")

			result, err := checkErr(resp, originalErr)

			Expect(result).To(Equal(resp))
			Expect(err).To(MatchError("value error: original error"))
		})

		It("should handle value types with empty error", func() {
			resp := testValueErrorResponse{errorMsg: ""}
			originalErr := errors.New("original error")

			result, err := checkErr(resp, originalErr)

			Expect(result).To(Equal(resp))
			Expect(err).To(Equal(originalErr))
		})
	})
})

// Test helper types
type testErrorResponse struct {
	errorMsg string
}

func (t *testErrorResponse) GetError() string {
	if t == nil {
		return "" // This is what would typically happen with a typed nil
	}
	return t.errorMsg
}

type testNonErrorResponse struct {
	data string
}

type testValueErrorResponse struct {
	errorMsg string
}

func (t testValueErrorResponse) GetError() string {
	return t.errorMsg
}
