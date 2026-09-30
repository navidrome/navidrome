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
		tmpDir, err = os.MkdirTemp("", "ndpgen-test-*")
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
	Call(ctx context.Context, uri string) (responseJSON string, err error)
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
			Expect(m.Returns[0].Name).To(Equal("responseJSON"))
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

		It("should resolve structs defined in a sibling file of the same package", func() {
			// types.go defines Track and Artist — no host service here
			typesSrc := `package host

// Artist is a track participant.
type Artist struct {
	// ID is the artist identifier.
	ID   string ` + "`json:\"id\"`" + `
	// Name is the artist name.
	Name string ` + "`json:\"name\"`" + `
}

// Track is a media file projection.
type Track struct {
	// ID is the track identifier.
	ID   string ` + "`json:\"id\"`" + `
	// Title is the track title.
	Title string ` + "`json:\"title\"`" + `
	// Participants maps role to artists (transitive dependency test).
	Participants map[string][]Artist ` + "`json:\"participants\"`" + `
}
`
			// service.go defines the host service that references Track from types.go
			serviceSrc := `package host

import "context"

// MatcherService matches tracks.
//nd:hostservice name=Matcher permission=matcher
type MatcherService interface {
	// MatchTrack finds a matching track.
	//nd:hostfunc
	MatchTrack(ctx context.Context, t Track) (matched bool, err error)
}
`
			err := os.WriteFile(filepath.Join(tmpDir, "types.go"), []byte(typesSrc), 0600)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(filepath.Join(tmpDir, "service.go"), []byte(serviceSrc), 0600)
			Expect(err).NotTo(HaveOccurred())

			services, err := ParseDirectory(tmpDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(services).To(HaveLen(1))

			svc := services[0]
			Expect(svc.Name).To(Equal("Matcher"))
			Expect(svc.Methods).To(HaveLen(1))
			Expect(svc.Methods[0].Name).To(Equal("MatchTrack"))

			// Track must be resolved from the sibling file, not just the service file.
			// Artist must also be transitively resolved (Track.Participants references it).
			structNames := make([]string, len(svc.Structs))
			for i, s := range svc.Structs {
				structNames[i] = s.Name
			}
			Expect(structNames).To(ConsistOf("Track", "Artist"))
		})

		It("returns an error when a shared-type alias cannot be resolved (no registry)", func() {
			fileA := `package host

import "github.com/navidrome/navidrome/plugins/types"

type TrackInfo = types.TrackInfo
`
			fileB := `package host

import "context"

//nd:hostservice name=Matcher permission=matcher
type MatcherService interface {
	//nd:hostfunc
	Match(ctx context.Context, t TrackInfo) (bool, error)
}
`
			Expect(os.WriteFile(filepath.Join(tmpDir, "aliases.go"), []byte(fileA), 0600)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(tmpDir, "matcher.go"), []byte(fileB), 0600)).To(Succeed())

			_, err := ParseDirectoryWithShared(tmpDir, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("TrackInfo"))
			Expect(err.Error()).To(ContainSubstring("-shared"))
		})

		It("resolves shared-type aliases declared in a sibling file (package-wide alias map)", func() {
			// File A: declares the shared-type alias in the same package
			fileA := `package host

import "github.com/navidrome/navidrome/plugins/types"

// Deprecated: use types.Track.
type Track = types.Track
`
			// File B: declares the host service that references Track from file A
			fileB := `package host

import "context"

//nd:hostservice name=Matcher permission=matcher
type MatcherService interface {
	//nd:hostfunc
	MatchSongs(ctx context.Context, query string) (results []Track, err error)
}
`
			Expect(os.WriteFile(filepath.Join(tmpDir, "aliases.go"), []byte(fileA), 0600)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(tmpDir, "matcher.go"), []byte(fileB), 0600)).To(Succeed())

			shared := map[string]StructDef{
				"Track": {
					Name: "Track",
					Fields: []FieldDef{
						{Name: "Title", Type: "string", JSONTag: "title"},
						{Name: "Artist", Type: "string", JSONTag: "artist"},
					},
				},
			}

			services, err := ParseDirectoryWithShared(tmpDir, shared)
			Expect(err).NotTo(HaveOccurred())
			Expect(services).To(HaveLen(1))

			byName := map[string]SharedAlias{}
			for _, a := range services[0].SharedAliases {
				byName[a.Name] = a
			}
			// Track alias is in a sibling file — must be resolved package-wide
			Expect(byName).To(HaveKey("Track"))
			Expect(byName["Track"].Target).To(Equal("types.Track"))
		})
	})

	Describe("stream types", func() {
		parseService := func(src string) ([]Service, error) {
			Expect(os.WriteFile(filepath.Join(tmpDir, "blob.go"), []byte(src), 0600)).To(Succeed())
			return ParseDirectory(tmpDir)
		}

		const header = `package host

import (
	"context"
	"io"
)
`
		const upload = `
type Upload struct {
	Name string
	Body io.ReadCloser
}
`

		It("accepts streams as direct params, results, and fields of param/result structs", func() {
			services, err := parseService(header + upload + `
//nd:hostservice name=Blob permission=blob
type BlobService interface {
	//nd:hostfunc
	Put(ctx context.Context, upload Upload, dst io.WriteCloser) (*Upload, error)
	//nd:hostfunc
	Open(ctx context.Context, name string) (io.ReadCloser, error)
}
`)
			Expect(err).NotTo(HaveOccurred())
			Expect(services).To(HaveLen(1))
			svc := services[0]
			Expect(svc.HasStreams()).To(BeTrue())
			Expect(svc.StreamStructs()).To(HaveLen(1))
			Expect(svc.StreamStructs()[0].Name).To(Equal("Upload"))

			put, open := svc.Methods[0], svc.Methods[1]
			Expect(put.Name).To(Equal("Put"))
			Expect(open.Name).To(Equal("Open"))

			st, isPtr, found := put.Params[0].StreamStruct(svc)
			Expect(found).To(BeTrue())
			Expect(isPtr).To(BeFalse())
			Expect(st.Name).To(Equal("Upload"))
			Expect(put.Params[0].StreamKind()).To(BeEmpty())
			Expect(put.Params[1].StreamKind()).To(Equal("writer"))
			_, _, found = put.Params[1].StreamStruct(svc)
			Expect(found).To(BeFalse())

			st, isPtr, found = put.Returns[0].StreamStruct(svc)
			Expect(found).To(BeTrue())
			Expect(isPtr).To(BeTrue())
			Expect(st.Name).To(Equal("Upload"))

			Expect(open.Returns[0].StreamKind()).To(Equal("reader"))
			_, _, found = open.Params[0].StreamStruct(svc)
			Expect(found).To(BeFalse())

			Expect(put.HasStreamParams(svc)).To(BeTrue())
			Expect(put.HasStreamReturns(svc)).To(BeTrue())
			Expect(put.HasStreams(svc)).To(BeTrue())
			Expect(open.HasStreamParams(svc)).To(BeFalse())
			Expect(open.HasStreamReturns(svc)).To(BeTrue())
			Expect(open.HasStreams(svc)).To(BeTrue())
		})

		It("reports no streams for a service without stream types", func() {
			services, err := parseService(header + `
type Item struct {
	Name string
}

//nd:hostservice name=Blob permission=blob
type BlobService interface {
	//nd:hostfunc
	Get(ctx context.Context, name string) (*Item, error)
}
`)
			Expect(err).NotTo(HaveOccurred())
			svc := services[0]
			Expect(svc.HasStreams()).To(BeFalse())
			Expect(svc.StreamStructs()).To(BeEmpty())
			Expect(svc.Structs[0].HasStreams()).To(BeFalse())
			Expect(svc.Methods[0].HasStreams(svc)).To(BeFalse())
		})

		DescribeTable("rejects plain io.Reader and io.Writer",
			func(src, want string) {
				_, err := parseService(header + src)
				Expect(err).To(MatchError(ContainSubstring("streams must be io.ReadCloser or io.WriteCloser, got " + want)))
				Expect(err).To(MatchError(ContainSubstring("parsing blob.go")))
			},
			Entry("as a param", `
//nd:hostservice name=Blob permission=blob
type BlobService interface {
	//nd:hostfunc
	Read(ctx context.Context, r io.Reader) error
}
`, "io.Reader"),
			Entry("as a struct field", `
type Sink struct {
	Out io.Writer
}

//nd:hostservice name=Blob permission=blob
type BlobService interface {
	//nd:hostfunc
	Write(ctx context.Context, sink Sink) error
}
`, "io.Writer"),
		)

		DescribeTable("rejects streams in slices, maps, and nested structs",
			func(src string) {
				_, err := parseService(header + src)
				Expect(err).To(MatchError(ContainSubstring("streams are only supported as direct parameters, results, or fields of a parameter or result struct")))
			},
			Entry("slice param", `
//nd:hostservice name=Blob permission=blob
type BlobService interface {
	//nd:hostfunc
	Put(ctx context.Context, items []io.ReadCloser) error
}
`),
			Entry("map field", `
type Multi struct {
	Parts map[string]io.ReadCloser
}

//nd:hostservice name=Blob permission=blob
type BlobService interface {
	//nd:hostfunc
	Put(ctx context.Context, multi Multi) error
}
`),
			Entry("nested struct", upload+`
type Outer struct {
	In Upload
}

//nd:hostservice name=Blob permission=blob
type BlobService interface {
	//nd:hostfunc
	Put(ctx context.Context, outer Outer) error
}
`),
			Entry("slice of stream structs", upload+`
//nd:hostservice name=Blob permission=blob
type BlobService interface {
	//nd:hostfunc
	List(ctx context.Context) ([]Upload, error)
}
`),
		)

		It("rejects a stream method without an error result", func() {
			_, err := parseService(header + `
//nd:hostservice name=Blob permission=blob
type BlobService interface {
	//nd:hostfunc
	Drain(ctx context.Context, in io.ReadCloser) string
}
`)
			Expect(err).To(MatchError(ContainSubstring("host functions with streams must return an error")))
		})

		DescribeTable("rejects parameter and result names reserved by generated code",
			func(src, name string) {
				_, err := parseService(header + src)
				Expect(err).To(MatchError(ContainSubstring(`parameter name "` + name + `" is reserved by generated code`)))
			},
			Entry("stream struct param", upload+`
//nd:hostservice name=Blob permission=blob
type BlobService interface {
	//nd:hostfunc
	Put(ctx context.Context, req Upload) error
}
`, "req"),
			Entry("named stream result", `
//nd:hostservice name=Blob permission=blob
type BlobService interface {
	//nd:hostfunc
	Open(ctx context.Context, name string) (conv io.ReadCloser, err error)
}
`, "conv"),
			Entry("plain method", `
//nd:hostservice name=Blob permission=blob
type BlobService interface {
	//nd:hostfunc
	Get(ctx context.Context, service string) (response string, err error)
}
`, "service"),
		)

		Describe("shared types", func() {
			var shared map[string]StructDef

			BeforeEach(func() {
				sharedDir := filepath.Join(tmpDir, "types")
				Expect(os.Mkdir(sharedDir, 0700)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(sharedDir, "blob.go"), []byte(`package types

import "io"

type Blob struct {
	Body io.ReadCloser
}
`), 0600)).To(Succeed())
				var err error
				shared, err = LoadSharedTypes(sharedDir)
				Expect(err).NotTo(HaveOccurred())
			})

			DescribeTable("rejects stream fields in shared types used by a host service",
				func(src string) {
					Expect(os.WriteFile(filepath.Join(tmpDir, "blob.go"), []byte(`package host

import (
	"context"

	"github.com/navidrome/navidrome/plugins/types"
)
`+src), 0600)).To(Succeed())

					_, err := ParseDirectoryWithShared(tmpDir, shared)
					Expect(err).To(MatchError(ContainSubstring("streams are not supported in shared types")))
				},
				Entry("as a direct param", `
//nd:hostservice name=Blob permission=blob
type BlobService interface {
	//nd:hostfunc
	Put(ctx context.Context, blob types.Blob) error
}
`),
				Entry("through a local struct field", `
type Wrapper struct {
	B types.Blob
}

//nd:hostservice name=Blob permission=blob
type BlobService interface {
	//nd:hostfunc
	Put(ctx context.Context, w Wrapper) error
}
`),
				Entry("through a shared alias in a local struct field", `
type Blob = types.Blob

type Wrapper struct {
	B Blob
}

//nd:hostservice name=Blob permission=blob
type BlobService interface {
	//nd:hostfunc
	Put(ctx context.Context, w Wrapper) error
}
`),
			)

			It("rejects stream fields in shared types a capability reaches through a local struct", func() {
				Expect(os.WriteFile(filepath.Join(tmpDir, "upload.go"), []byte(`package capabilities

import "github.com/navidrome/navidrome/plugins/types"

//nd:capability name=upload
type Uploader interface {
	//nd:export name=nd_upload
	Upload(UploadInput) error
}

type UploadInput struct {
	Blob types.Blob `+"`json:\"blob\"`"+`
}
`), 0600)).To(Succeed())

				_, err := ParseCapabilitiesWithShared(tmpDir, shared)
				Expect(err).To(MatchError(ContainSubstring("streams are not supported in shared types")))
			})
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
			// Host-side types are public
			Expect(m.RequestTypeName("SubsonicAPI")).To(Equal("SubsonicAPICallRequest"))
			Expect(m.ResponseTypeName("SubsonicAPI")).To(Equal("SubsonicAPICallResponse"))
			// Client/PDK types are private
			Expect(m.ClientRequestTypeName("SubsonicAPI")).To(Equal("subsonicAPICallRequest"))
			Expect(m.ClientResponseTypeName("SubsonicAPI")).To(Equal("subsonicAPICallResponse"))
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

	Describe("ParseCapabilities", func() {
		It("should parse a simple capability interface", func() {
			src := `package capabilities

// MetadataAgent provides metadata retrieval.
//nd:capability name=metadata
type MetadataAgent interface {
	// GetArtistBiography returns artist biography.
	//nd:export name=nd_get_artist_biography
	GetArtistBiography(ArtistInput) (ArtistBiographyOutput, error)
}

// ArtistInput is the input for artist-related functions.
type ArtistInput struct {
	// ID is the artist ID.
	ID string ` + "`json:\"id\"`" + `
	// Name is the artist name.
	Name string ` + "`json:\"name\"`" + `
}

// ArtistBiographyOutput is the output for GetArtistBiography.
type ArtistBiographyOutput struct {
	// Biography is the biography text.
	Biography string ` + "`json:\"biography\"`" + `
}
`
			err := os.WriteFile(filepath.Join(tmpDir, "metadata.go"), []byte(src), 0600)
			Expect(err).NotTo(HaveOccurred())

			capabilities, err := ParseCapabilities(tmpDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(capabilities).To(HaveLen(1))

			cap := capabilities[0]
			Expect(cap.Name).To(Equal("metadata"))
			Expect(cap.Interface).To(Equal("MetadataAgent"))
			Expect(cap.Required).To(BeFalse())
			Expect(cap.Doc).To(ContainSubstring("MetadataAgent provides metadata retrieval"))
			Expect(cap.Methods).To(HaveLen(1))

			m := cap.Methods[0]
			Expect(m.Name).To(Equal("GetArtistBiography"))
			Expect(m.ExportName).To(Equal("nd_get_artist_biography"))
			Expect(m.Input.Type).To(Equal("ArtistInput"))
			Expect(m.Output.Type).To(Equal("ArtistBiographyOutput"))

			// Check structs were collected
			Expect(cap.Structs).To(HaveLen(2))
		})

		It("should parse a required capability", func() {
			src := `package capabilities

// Scrobbler requires all methods to be implemented.
//nd:capability name=scrobbler required=true
type Scrobbler interface {
	//nd:export name=nd_scrobbler_is_authorized
	IsAuthorized(AuthInput) (AuthOutput, error)

	//nd:export name=nd_scrobbler_scrobble
	Scrobble(ScrobbleInput) (ScrobblerOutput, error)
}

type AuthInput struct {
	UserID string ` + "`json:\"userId\"`" + `
}

type AuthOutput struct {
	Authorized bool ` + "`json:\"authorized\"`" + `
}

type ScrobbleInput struct {
	UserID string ` + "`json:\"userId\"`" + `
}

type ScrobblerOutput struct {
	Error *string ` + "`json:\"error,omitempty\"`" + `
}
`
			err := os.WriteFile(filepath.Join(tmpDir, "scrobbler.go"), []byte(src), 0600)
			Expect(err).NotTo(HaveOccurred())

			capabilities, err := ParseCapabilities(tmpDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(capabilities).To(HaveLen(1))

			cap := capabilities[0]
			Expect(cap.Name).To(Equal("scrobbler"))
			Expect(cap.Required).To(BeTrue())
			Expect(cap.Methods).To(HaveLen(2))
		})

		It("should parse type aliases and consts", func() {
			src := `package capabilities

//nd:capability name=scrobbler required=true
type Scrobbler interface {
	//nd:export name=nd_scrobble
	Scrobble(ScrobbleInput) (ScrobblerOutput, error)
}

type ScrobbleInput struct {
	UserID string ` + "`json:\"userId\"`" + `
}

// ScrobblerErrorType indicates error handling behavior.
type ScrobblerErrorType string

const (
	// ScrobblerErrorNone indicates no error.
	ScrobblerErrorNone ScrobblerErrorType = "none"
	// ScrobblerErrorRetry indicates retry later.
	ScrobblerErrorRetry ScrobblerErrorType = "retry"
)

type ScrobblerOutput struct {
	ErrorType *ScrobblerErrorType ` + "`json:\"errorType,omitempty\"`" + `
}
`
			err := os.WriteFile(filepath.Join(tmpDir, "scrobbler.go"), []byte(src), 0600)
			Expect(err).NotTo(HaveOccurred())

			capabilities, err := ParseCapabilities(tmpDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(capabilities).To(HaveLen(1))

			cap := capabilities[0]
			// Type alias should be collected
			Expect(cap.TypeAliases).To(HaveLen(1))
			Expect(cap.TypeAliases[0].Name).To(Equal("ScrobblerErrorType"))
			Expect(cap.TypeAliases[0].Type).To(Equal("string"))

			// Consts should be collected
			Expect(cap.Consts).To(HaveLen(1))
			Expect(cap.Consts[0].Type).To(Equal("ScrobblerErrorType"))
			Expect(cap.Consts[0].Values).To(HaveLen(2))
			Expect(cap.Consts[0].Values[0].Name).To(Equal("ScrobblerErrorNone"))
			Expect(cap.Consts[0].Values[0].Value).To(Equal(`"none"`))
		})

		It("should collect nested struct dependencies", func() {
			src := `package capabilities

//nd:capability name=metadata
type MetadataAgent interface {
	//nd:export name=nd_get_images
	GetImages(ArtistInput) (ImagesOutput, error)
}

type ArtistInput struct {
	ID string ` + "`json:\"id\"`" + `
}

type ImagesOutput struct {
	Images []ImageInfo ` + "`json:\"images\"`" + `
}

type ImageInfo struct {
	URL string ` + "`json:\"url\"`" + `
	Size int32 ` + "`json:\"size\"`" + `
}
`
			err := os.WriteFile(filepath.Join(tmpDir, "metadata.go"), []byte(src), 0600)
			Expect(err).NotTo(HaveOccurred())

			capabilities, err := ParseCapabilities(tmpDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(capabilities).To(HaveLen(1))

			cap := capabilities[0]
			// Should collect all 3 structs: ArtistInput, ImagesOutput, and ImageInfo
			Expect(cap.Structs).To(HaveLen(3))

			structNames := make([]string, len(cap.Structs))
			for i, s := range cap.Structs {
				structNames[i] = s.Name
			}
			Expect(structNames).To(ContainElements("ArtistInput", "ImagesOutput", "ImageInfo"))
		})

		It("should return empty slice for directory with no capabilities", func() {
			src := `package capabilities

type RegularInterface interface {
	Method() error
}
`
			err := os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte(src), 0600)
			Expect(err).NotTo(HaveOccurred())

			capabilities, err := ParseCapabilities(tmpDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(capabilities).To(BeEmpty())
		})

		It("should ignore methods without export annotation", func() {
			src := `package capabilities

//nd:capability name=test
type TestCapability interface {
	//nd:export name=nd_exported
	ExportedMethod(Input) (Output, error)

	// This method has no export annotation
	NotExportedMethod(Input) (Output, error)
}

type Input struct {
	Value string ` + "`json:\"value\"`" + `
}

type Output struct {
	Result string ` + "`json:\"result\"`" + `
}
`
			err := os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte(src), 0600)
			Expect(err).NotTo(HaveOccurred())

			capabilities, err := ParseCapabilities(tmpDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(capabilities).To(HaveLen(1))

			// Only the exported method should be captured
			Expect(capabilities[0].Methods).To(HaveLen(1))
			Expect(capabilities[0].Methods[0].Name).To(Equal("ExportedMethod"))
		})

		It("distinguishes Go type aliases from defined types", func() {
			src := `package capabilities

import "github.com/navidrome/navidrome/plugins/types"

// Deprecated: use types.ArtistRef.
type ArtistRef = types.ArtistRef

// ScrobblerError is a sentinel error string.
type ScrobblerError string

//nd:capability name=scrobbler required=true
type Scrobbler interface {
	//nd:export name=nd_scrobbler_check
	Check(ArtistRef) (bool, error)
}
`
			Expect(os.WriteFile(filepath.Join(tmpDir, "scrobbler.go"), []byte(src), 0600)).To(Succeed())

			shared := map[string]StructDef{
				"ArtistRef": {Name: "ArtistRef", Fields: []FieldDef{{Name: "Name", Type: "string", JSONTag: "name"}}},
			}
			caps, err := ParseCapabilitiesWithShared(tmpDir, shared)
			Expect(err).NotTo(HaveOccurred())
			Expect(caps).To(HaveLen(1))

			// ArtistRef is a shared-type alias (types.*): it lands in SharedAliases, not TypeAliases.
			sharedByName := map[string]SharedAlias{}
			for _, a := range caps[0].SharedAliases {
				sharedByName[a.Name] = a
			}
			Expect(sharedByName).To(HaveKey("ArtistRef"))
			Expect(sharedByName["ArtistRef"].Target).To(Equal("types.ArtistRef"))

			// ScrobblerError is a plain defined type: it stays in TypeAliases.
			typeByName := map[string]TypeAlias{}
			for _, a := range caps[0].TypeAliases {
				typeByName[a.Name] = a
			}
			Expect(typeByName).To(HaveKey("ScrobblerError"))
			Expect(typeByName["ScrobblerError"].IsAlias).To(BeFalse())
		})

		It("rejects stream types in capability exports", func() {
			src := `package capabilities

import "io"

//nd:capability name=upload
type Uploader interface {
	//nd:export name=nd_upload
	Upload(UploadInput) (UploadOutput, error)
}

type UploadInput struct {
	Body io.ReadCloser ` + "`json:\"body\"`" + `
}

type UploadOutput struct {
	OK bool ` + "`json:\"ok\"`" + `
}
`
			Expect(os.WriteFile(filepath.Join(tmpDir, "upload.go"), []byte(src), 0600)).To(Succeed())

			_, err := ParseCapabilities(tmpDir)
			Expect(err).To(MatchError(ContainSubstring("streams are not supported in capabilities")))
		})
	})

	Describe("ParseCapabilitiesWithShared", func() {
		It("returns an error when a shared-type alias cannot be resolved (no registry)", func() {
			src := `package capabilities

import "github.com/navidrome/navidrome/plugins/types"

// TrackInfo is an alias for the shared type.
type TrackInfo = types.TrackInfo

//nd:capability name=nowplaying required=true
type NowPlaying interface {
	//nd:export name=nd_now_playing
	NowPlaying(TrackInfo) error
}
`
			Expect(os.WriteFile(filepath.Join(tmpDir, "nowplaying.go"), []byte(src), 0600)).To(Succeed())

			_, err := ParseCapabilitiesWithShared(tmpDir, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("TrackInfo"))
			Expect(err.Error()).To(ContainSubstring("-shared"))
		})

		It("resolves shared-type aliases against the registry", func() {
			shared := map[string]StructDef{
				"ArtistRef": {Name: "ArtistRef", Fields: []FieldDef{{Name: "Name", Type: "string", JSONTag: "name"}}},
				"TrackInfo": {Name: "TrackInfo", Fields: []FieldDef{
					{Name: "Title", Type: "string", JSONTag: "title"},
					{Name: "Artists", Type: "[]ArtistRef", JSONTag: "artists"},
				}},
			}
			src := `package capabilities

import "github.com/navidrome/navidrome/plugins/types"

// Deprecated: use types.TrackInfo.
type TrackInfo = types.TrackInfo

// Deprecated: use types.ArtistRef.
type ArtistRef = types.ArtistRef

// NowPlayingRequest carries a track.
type NowPlayingRequest struct {
	Track TrackInfo ` + "`json:\"track\"`" + `
}

//nd:capability name=scrobbler required=true
type Scrobbler interface {
	//nd:export name=nd_scrobbler_now_playing
	NowPlaying(NowPlayingRequest) error
}
`
			Expect(os.WriteFile(filepath.Join(tmpDir, "scrobbler.go"), []byte(src), 0600)).To(Succeed())

			caps, err := ParseCapabilitiesWithShared(tmpDir, shared)
			Expect(err).NotTo(HaveOccurred())
			Expect(caps).To(HaveLen(1))

			names := []string{}
			for _, a := range caps[0].SharedAliases {
				names = append(names, a.Name)
			}
			// TrackInfo is referenced directly; ArtistRef is pulled in transitively via TrackInfo.Artists.
			Expect(names).To(ContainElements("TrackInfo", "ArtistRef"))

			byName := map[string]SharedAlias{}
			for _, a := range caps[0].SharedAliases {
				byName[a.Name] = a
			}
			Expect(byName["TrackInfo"].Target).To(Equal("types.TrackInfo"))
			Expect(byName["TrackInfo"].Def.Fields).To(HaveLen(2)) // for schema inlining
			Expect(byName["ArtistRef"].Target).To(Equal("types.ArtistRef"))
		})

		It("resolves shared types from qualified types.X references with a renamed alias", func() {
			shared := map[string]StructDef{
				"ArtistRef": {Name: "ArtistRef", Fields: []FieldDef{{Name: "Name", Type: "string", JSONTag: "name"}}},
				"Track": {Name: "Track", Fields: []FieldDef{
					{Name: "Title", Type: "string", JSONTag: "title"},
					{Name: "Artists", Type: "[]ArtistRef", JSONTag: "artists"},
				}},
			}
			src := `package capabilities

import "github.com/navidrome/navidrome/plugins/types"

// Deprecated: use types.Track.
type TrackInfo = types.Track

// Deprecated: use types.ArtistRef.
type ArtistRef = types.ArtistRef

// NowPlayingRequest carries a track.
type NowPlayingRequest struct {
	Track types.Track ` + "`json:\"track\"`" + `
}

//nd:capability name=scrobbler required=true
type Scrobbler interface {
	//nd:export name=nd_scrobbler_now_playing
	NowPlaying(NowPlayingRequest) error
}
`
			Expect(os.WriteFile(filepath.Join(tmpDir, "scrobbler.go"), []byte(src), 0600)).To(Succeed())

			caps, err := ParseCapabilitiesWithShared(tmpDir, shared)
			Expect(err).NotTo(HaveOccurred())
			Expect(caps).To(HaveLen(1))

			byName := map[string]SharedAlias{}
			for _, a := range caps[0].SharedAliases {
				byName[a.Name] = a
			}
			// The deprecated alias keeps its name (TrackInfo) but now targets types.Track.
			// ArtistRef is pulled in transitively via Track.Artists.
			Expect(byName).To(HaveKey("TrackInfo"))
			Expect(byName).To(HaveKey("ArtistRef"))
			Expect(byName["TrackInfo"].Target).To(Equal("types.Track"))
			Expect(byName["TrackInfo"].Def.Fields).To(HaveLen(2)) // for schema inlining
			Expect(byName["ArtistRef"].Target).To(Equal("types.ArtistRef"))

			// The capability struct field keeps the canonical qualified reference.
			var nowPlaying StructDef
			for _, st := range caps[0].Structs {
				if st.Name == "NowPlayingRequest" {
					nowPlaying = st
				}
			}
			Expect(nowPlaying.Fields).To(HaveLen(1))
			Expect(nowPlaying.Fields[0].Type).To(Equal("types.Track"))
		})
	})

	Describe("Export helpers", func() {
		It("should generate correct provider interface name", func() {
			e := Export{Name: "GetArtistBiography"}
			Expect(e.ProviderInterfaceName()).To(Equal("ArtistBiographyProvider"))

			e = Export{Name: "OnInit"}
			Expect(e.ProviderInterfaceName()).To(Equal("InitProvider"))
		})

		It("should generate correct impl variable name", func() {
			e := Export{Name: "GetArtistBiography"}
			Expect(e.ImplVarName()).To(Equal("artistBiographyImpl"))

			e = Export{Name: "OnInit"}
			Expect(e.ImplVarName()).To(Equal("initImpl"))
		})

		It("should generate correct export function name", func() {
			e := Export{Name: "GetArtistBiography", ExportName: "nd_get_artist_biography"}
			Expect(e.ExportFuncName()).To(Equal("_NdGetArtistBiography"))
		})
	})
})
