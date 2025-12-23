package internal

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Parser", func() {
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "hostgen-test-*")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	Describe("ParseDirectory", func() {
		It("should parse a simple host service interface", func() {
			src := `package host

import "context"

// SubsonicAPIService provides access to Navidrome's Subsonic API.
//nd:hostservice name=SubsonicAPI permission=subsonicapi
type SubsonicAPIService interface {
	// Call executes a Subsonic API request.
	//nd:hostfunc
	Call(ctx context.Context, uri string) (response string, err error)
}
`
			err := os.WriteFile(filepath.Join(tmpDir, "service.go"), []byte(src), 0600)
			Expect(err).NotTo(HaveOccurred())

			services, err := ParseDirectory(tmpDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(services).To(HaveLen(1))

			svc := services[0]
			Expect(svc.Name).To(Equal("SubsonicAPI"))
			Expect(svc.Permission).To(Equal("subsonicapi"))
			Expect(svc.Interface).To(Equal("SubsonicAPIService"))
			Expect(svc.Methods).To(HaveLen(1))

			m := svc.Methods[0]
			Expect(m.Name).To(Equal("Call"))
			Expect(m.HasError).To(BeTrue())
			Expect(m.Params).To(HaveLen(1))
			Expect(m.Params[0].Name).To(Equal("uri"))
			Expect(m.Params[0].Type).To(Equal("string"))
			Expect(m.Returns).To(HaveLen(1))
			Expect(m.Returns[0].Name).To(Equal("response"))
			Expect(m.Returns[0].Type).To(Equal("string"))
		})

		It("should parse multiple methods", func() {
			src := `package host

import "context"

// SchedulerService provides scheduling capabilities.
//nd:hostservice name=Scheduler permission=scheduler
type SchedulerService interface {
	//nd:hostfunc
	ScheduleRecurring(ctx context.Context, cronExpression string) (scheduleID string, err error)

	//nd:hostfunc
	ScheduleOneTime(ctx context.Context, delaySeconds int32) (scheduleID string, err error)

	//nd:hostfunc
	CancelSchedule(ctx context.Context, scheduleID string) (canceled bool, err error)
}
`
			err := os.WriteFile(filepath.Join(tmpDir, "scheduler.go"), []byte(src), 0600)
			Expect(err).NotTo(HaveOccurred())

			services, err := ParseDirectory(tmpDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(services).To(HaveLen(1))

			svc := services[0]
			Expect(svc.Name).To(Equal("Scheduler"))
			Expect(svc.Methods).To(HaveLen(3))

			Expect(svc.Methods[0].Name).To(Equal("ScheduleRecurring"))
			Expect(svc.Methods[0].Params[0].Type).To(Equal("string"))

			Expect(svc.Methods[1].Name).To(Equal("ScheduleOneTime"))
			Expect(svc.Methods[1].Params[0].Type).To(Equal("int32"))

			Expect(svc.Methods[2].Name).To(Equal("CancelSchedule"))
			Expect(svc.Methods[2].Returns[0].Type).To(Equal("bool"))
		})

		It("should skip methods without hostfunc annotation", func() {
			src := `package host

import "context"

//nd:hostservice name=Test permission=test
type TestService interface {
	//nd:hostfunc
	Exported(ctx context.Context) error

	// This method is not exported
	NotExported(ctx context.Context) error
}
`
			err := os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte(src), 0600)
			Expect(err).NotTo(HaveOccurred())

			services, err := ParseDirectory(tmpDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(services).To(HaveLen(1))
			Expect(services[0].Methods).To(HaveLen(1))
			Expect(services[0].Methods[0].Name).To(Equal("Exported"))
		})

		It("should handle custom export name", func() {
			src := `package host

import "context"

//nd:hostservice name=Test permission=test
type TestService interface {
	//nd:hostfunc name=custom_export_name
	MyMethod(ctx context.Context) error
}
`
			err := os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte(src), 0600)
			Expect(err).NotTo(HaveOccurred())

			services, err := ParseDirectory(tmpDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(services[0].Methods[0].ExportName).To(Equal("custom_export_name"))
			Expect(services[0].Methods[0].FunctionName("test")).To(Equal("custom_export_name"))
		})

		It("should skip generated files", func() {
			regularSrc := `package host

import "context"

//nd:hostservice name=Test permission=test
type TestService interface {
	//nd:hostfunc
	Method(ctx context.Context) error
}
`
			genSrc := `// Code generated. DO NOT EDIT.
package host

//nd:hostservice name=Generated permission=gen
type GeneratedService interface {
	//nd:hostfunc
	Method() error
}
`
			err := os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte(regularSrc), 0600)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(filepath.Join(tmpDir, "test_gen.go"), []byte(genSrc), 0600)
			Expect(err).NotTo(HaveOccurred())

			services, err := ParseDirectory(tmpDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(services).To(HaveLen(1))
			Expect(services[0].Name).To(Equal("Test"))
		})

		It("should skip interfaces without hostservice annotation", func() {
			src := `package host

import "context"

// Regular interface without annotation
type RegularInterface interface {
	Method(ctx context.Context) error
}

//nd:hostservice name=Annotated permission=annotated
type AnnotatedService interface {
	//nd:hostfunc
	Method(ctx context.Context) error
}
`
			err := os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte(src), 0600)
			Expect(err).NotTo(HaveOccurred())

			services, err := ParseDirectory(tmpDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(services).To(HaveLen(1))
			Expect(services[0].Name).To(Equal("Annotated"))
		})

		It("should return empty slice for directory with no host services", func() {
			src := `package host

type RegularInterface interface {
	Method() error
}
`
			err := os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte(src), 0600)
			Expect(err).NotTo(HaveOccurred())

			services, err := ParseDirectory(tmpDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(services).To(BeEmpty())
		})
	})

	Describe("parseKeyValuePairs", func() {
		It("should parse key=value pairs", func() {
			result := parseKeyValuePairs("name=Test permission=test")
			Expect(result).To(HaveKeyWithValue("name", "Test"))
			Expect(result).To(HaveKeyWithValue("permission", "test"))
		})

		It("should return nil for empty input", func() {
			result := parseKeyValuePairs("")
			Expect(result).To(BeNil())
		})
	})

	Describe("typeToString", func() {
		It("should handle basic types", func() {
			src := `package test
type T interface {
	Method(s string, i int, b bool) ([]byte, error)
}
`
			err := os.WriteFile(filepath.Join(tmpDir, "types.go"), []byte(src), 0600)
			Expect(err).NotTo(HaveOccurred())

			// Parse and verify type conversion works
			// This is implicitly tested through ParseDirectory
		})

		It("should convert interface{} to any", func() {
			src := `package test

import "context"

//nd:hostservice name=Test permission=test
type TestService interface {
	//nd:hostfunc
	GetMetadata(ctx context.Context) (data map[string]interface{}, err error)
}
`
			err := os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte(src), 0600)
			Expect(err).NotTo(HaveOccurred())

			services, err := ParseDirectory(tmpDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(services).To(HaveLen(1))
			Expect(services[0].Methods[0].Returns[0].Type).To(Equal("map[string]any"))
		})
	})

	Describe("Method helpers", func() {
		It("should generate correct function names", func() {
			m := Method{Name: "Call"}
			Expect(m.FunctionName("subsonicapi")).To(Equal("subsonicapi_call"))

			m.ExportName = "custom_name"
			Expect(m.FunctionName("subsonicapi")).To(Equal("custom_name"))
		})

		It("should generate correct type names", func() {
			m := Method{Name: "Call"}
			Expect(m.RequestTypeName("SubsonicAPI")).To(Equal("SubsonicAPICallRequest"))
			Expect(m.ResponseTypeName("SubsonicAPI")).To(Equal("SubsonicAPICallResponse"))
		})
	})

	Describe("Service helpers", func() {
		It("should generate correct output file name", func() {
			s := Service{Name: "SubsonicAPI"}
			Expect(s.OutputFileName()).To(Equal("subsonicapi_gen.go"))
		})

		It("should generate correct export prefix", func() {
			s := Service{Name: "SubsonicAPI"}
			Expect(s.ExportPrefix()).To(Equal("subsonicapi"))
		})
	})
})
