package plugins

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/plugins/api"
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
		It("should return nil error when both resp and err are nil", func() {
			var resp *testErrorResponse

			result, err := checkErr(resp, nil)

			Expect(result).To(BeNil())
			Expect(err).To(BeNil())
		})

		It("should return original error unchanged for non-API errors", func() {
			var resp *testErrorResponse
			originalErr := errors.New("original error")

			result, err := checkErr(resp, originalErr)

			Expect(result).To(BeNil())
			Expect(err).To(Equal(originalErr))
		})

		It("should return mapped API error for ErrNotImplemented", func() {
			var resp *testErrorResponse
			err := errors.New("plugin:not_implemented")

			result, mappedErr := checkErr(resp, err)

			Expect(result).To(BeNil())
			Expect(mappedErr).To(Equal(api.ErrNotImplemented))
		})

		It("should return mapped API error for ErrNotFound", func() {
			var resp *testErrorResponse
			err := errors.New("plugin:not_found")

			result, mappedErr := checkErr(resp, err)

			Expect(result).To(BeNil())
			Expect(mappedErr).To(Equal(api.ErrNotFound))
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
			Expect(err).To(HaveOccurred())
			// Check that both error messages are present in the joined error
			errStr := err.Error()
			Expect(errStr).To(ContainSubstring("plugin error"))
			Expect(errStr).To(ContainSubstring("original error"))
		})

		It("should return mapped API error for ErrNotImplemented when no original error", func() {
			resp := &testErrorResponse{errorMsg: "plugin:not_implemented"}

			result, err := checkErr(resp, nil)

			Expect(result).To(Equal(resp))
			Expect(err).To(MatchError(api.ErrNotImplemented))
		})

		It("should return mapped API error for ErrNotFound when no original error", func() {
			resp := &testErrorResponse{errorMsg: "plugin:not_found"}

			result, err := checkErr(resp, nil)

			Expect(result).To(Equal(resp))
			Expect(err).To(MatchError(api.ErrNotFound))
		})

		It("should return mapped API error for ErrNotImplemented even with original error", func() {
			resp := &testErrorResponse{errorMsg: "plugin:not_implemented"}
			originalErr := errors.New("original error")

			result, err := checkErr(resp, originalErr)

			Expect(result).To(Equal(resp))
			Expect(err).To(MatchError(api.ErrNotImplemented))
		})

		It("should return mapped API error for ErrNotFound even with original error", func() {
			resp := &testErrorResponse{errorMsg: "plugin:not_found"}
			originalErr := errors.New("original error")

			result, err := checkErr(resp, originalErr)

			Expect(result).To(Equal(resp))
			Expect(err).To(MatchError(api.ErrNotFound))
		})
	})

	Context("when resp implements errorResponse with empty error", func() {
		It("should return original error unchanged", func() {
			resp := &testErrorResponse{errorMsg: ""}
			originalErr := errors.New("original error")

			result, err := checkErr(resp, originalErr)

			Expect(result).To(Equal(resp))
			Expect(err).To(MatchError(originalErr))
		})

		It("should return nil error when both are empty/nil", func() {
			resp := &testErrorResponse{errorMsg: ""}

			result, err := checkErr(resp, nil)

			Expect(result).To(Equal(resp))
			Expect(err).To(BeNil())
		})

		It("should map original API error when response error is empty", func() {
			resp := &testErrorResponse{errorMsg: ""}
			originalErr := errors.New("plugin:not_implemented")

			result, err := checkErr(resp, originalErr)

			Expect(result).To(Equal(resp))
			Expect(err).To(MatchError(api.ErrNotImplemented))
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

		It("should map original API error when response doesn't implement errorResponse", func() {
			resp := &testNonErrorResponse{data: "some data"}
			originalErr := errors.New("plugin:not_found")

			result, err := checkErr(resp, originalErr)

			Expect(result).To(Equal(resp))
			Expect(err).To(MatchError(api.ErrNotFound))
		})
	})

	Context("when resp is a value type (not pointer)", func() {
		It("should handle value types that implement errorResponse", func() {
			resp := testValueErrorResponse{errorMsg: "value error"}
			originalErr := errors.New("original error")

			result, err := checkErr(resp, originalErr)

			Expect(result).To(Equal(resp))
			Expect(err).To(HaveOccurred())
			// Check that both error messages are present in the joined error
			errStr := err.Error()
			Expect(errStr).To(ContainSubstring("value error"))
			Expect(errStr).To(ContainSubstring("original error"))
		})

		It("should handle value types with empty error", func() {
			resp := testValueErrorResponse{errorMsg: ""}
			originalErr := errors.New("original error")

			result, err := checkErr(resp, originalErr)

			Expect(result).To(Equal(resp))
			Expect(err).To(MatchError(originalErr))
		})

		It("should handle value types with API error", func() {
			resp := testValueErrorResponse{errorMsg: "plugin:not_implemented"}
			originalErr := errors.New("original error")

			result, err := checkErr(resp, originalErr)

			Expect(result).To(Equal(resp))
			Expect(err).To(MatchError(api.ErrNotImplemented))
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
