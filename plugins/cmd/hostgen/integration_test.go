package main

import (
	"go/format"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("hostgen CLI", Ordered, func() {
	var (
		testDir    string
		outputDir  string
		hostgenBin string
	)

	BeforeAll(func() {
		// Set testdata directory
		testdataDir = filepath.Join(mustGetWd(GinkgoT()), "plugins", "cmd", "hostgen", "testdata")

		// Build the hostgen binary
		hostgenBin = filepath.Join(os.TempDir(), "hostgen-test")
		cmd := exec.Command("go", "build", "-o", hostgenBin, ".")
		cmd.Dir = filepath.Join(mustGetWd(GinkgoT()), "plugins", "cmd", "hostgen")
		output, err := cmd.CombinedOutput()
		Expect(err).ToNot(HaveOccurred(), "Failed to build hostgen: %s", output)
		DeferCleanup(func() {
			os.Remove(hostgenBin)
		})
	})

	BeforeEach(func() {
		var err error
		testDir, err = os.MkdirTemp("", "hostgen-test-input-*")
		Expect(err).ToNot(HaveOccurred())
		outputDir, err = os.MkdirTemp("", "hostgen-test-output-*")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(testDir)
		os.RemoveAll(outputDir)
	})

	Describe("CLI flags and behavior", func() {
		BeforeEach(func() {
			serviceCode := `package testpkg

import "context"

//nd:hostservice name=Test permission=test
type TestService interface {
	//nd:hostfunc
	DoAction(ctx context.Context, input string) (output string, err error)
}
`
			Expect(os.WriteFile(filepath.Join(testDir, "service.go"), []byte(serviceCode), 0600)).To(Succeed())
		})

		It("supports verbose mode", func() {
			cmd := exec.Command(hostgenBin, "-input", testDir, "-output", outputDir, "-package", "testpkg", "-v")
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "Command failed: %s", output)

			outputStr := string(output)
			Expect(outputStr).To(ContainSubstring("Input directory:"))
			Expect(outputStr).To(ContainSubstring("Output directory:"))
			Expect(outputStr).To(ContainSubstring("Found 1 host service(s)"))
			Expect(outputStr).To(ContainSubstring("Generated"))
		})

		It("supports dry-run mode", func() {
			cmd := exec.Command(hostgenBin, "-input", testDir, "-output", outputDir, "-package", "testpkg", "-dry-run")
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "Command failed: %s", output)

			Expect(string(output)).To(ContainSubstring("RegisterTestHostFunctions"))
			Expect(filepath.Join(outputDir, "test_gen.go")).ToNot(BeAnExistingFile())
		})

		It("infers package name from output directory", func() {
			customOutput, err := os.MkdirTemp("", "mypkg")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(customOutput)

			cmd := exec.Command(hostgenBin, "-input", testDir, "-output", customOutput)
			_, err = cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())

			content, err := os.ReadFile(filepath.Join(customOutput, "test_gen.go"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("package mypkg"))
		})

		It("returns error for invalid input directory", func() {
			cmd := exec.Command(hostgenBin, "-input", "/nonexistent/path")
			output, err := cmd.CombinedOutput()
			Expect(err).To(HaveOccurred())
			Expect(string(output)).To(ContainSubstring("Error parsing source files"))
		})

		It("handles no annotated services gracefully", func() {
			Expect(os.WriteFile(filepath.Join(testDir, "service.go"), []byte("package testpkg\n"), 0600)).To(Succeed())

			cmd := exec.Command(hostgenBin, "-input", testDir, "-output", outputDir, "-v")
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "Command failed: %s", output)
			Expect(string(output)).To(ContainSubstring("No host services found"))
		})

		It("generates separate files for multiple services", func() {
			// Remove service.go created by BeforeEach
			Expect(os.Remove(filepath.Join(testDir, "service.go"))).To(Succeed())

			service1 := `package testpkg
import "context"
//nd:hostservice name=ServiceA permission=a
type ServiceA interface {
	//nd:hostfunc
	MethodA(ctx context.Context) error
}
`
			service2 := `package testpkg
import "context"
//nd:hostservice name=ServiceB permission=b
type ServiceB interface {
	//nd:hostfunc
	MethodB(ctx context.Context) error
}
`
			Expect(os.WriteFile(filepath.Join(testDir, "a.go"), []byte(service1), 0600)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(testDir, "b.go"), []byte(service2), 0600)).To(Succeed())

			cmd := exec.Command(hostgenBin, "-input", testDir, "-output", outputDir, "-package", "testpkg", "-v")
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "Command failed: %s", output)
			Expect(string(output)).To(ContainSubstring("Found 2 host service(s)"))

			Expect(filepath.Join(outputDir, "servicea_gen.go")).To(BeAnExistingFile())
			Expect(filepath.Join(outputDir, "serviceb_gen.go")).To(BeAnExistingFile())
		})

		It("generates only host code with -host-only flag", func() {
			cmd := exec.Command(hostgenBin, "-input", testDir, "-output", outputDir, "-package", "testpkg", "-host-only")
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "Command failed: %s", output)

			Expect(filepath.Join(outputDir, "test_gen.go")).To(BeAnExistingFile())
			Expect(filepath.Join(outputDir, "go")).ToNot(BeADirectory())
		})

		It("generates only client code with -plugin-only flag", func() {
			cmd := exec.Command(hostgenBin, "-input", testDir, "-output", outputDir, "-package", "main", "-plugin-only")
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "Command failed: %s", output)

			// Host code should not exist in output root
			entries, err := os.ReadDir(outputDir)
			Expect(err).ToNot(HaveOccurred())

			var genFiles []string
			for _, e := range entries {
				if e.Name() != "go" {
					genFiles = append(genFiles, e.Name())
				}
			}
			Expect(genFiles).To(BeEmpty(), "Expected no host code files, found: %v", genFiles)

			// Client code should exist in go/ subdirectory
			Expect(filepath.Join(outputDir, "go")).To(BeADirectory())
			Expect(filepath.Join(outputDir, "go", "nd_host_test.go")).To(BeAnExistingFile())
		})

		It("generates both host and client code by default", func() {
			cmd := exec.Command(hostgenBin, "-input", testDir, "-output", outputDir, "-package", "testpkg")
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "Command failed: %s", output)

			// Host code in output root
			Expect(filepath.Join(outputDir, "test_gen.go")).To(BeAnExistingFile())

			// Client code in go/ subdirectory
			Expect(filepath.Join(outputDir, "go")).To(BeADirectory())
			Expect(filepath.Join(outputDir, "go", "nd_host_test.go")).To(BeAnExistingFile())
		})

		It("rejects using both -host-only and -plugin-only together", func() {
			cmd := exec.Command(hostgenBin, "-input", testDir, "-output", outputDir, "-package", "testpkg", "-host-only", "-plugin-only")
			output, err := cmd.CombinedOutput()
			Expect(err).To(HaveOccurred())
			Expect(string(output)).To(ContainSubstring("-host-only and -plugin-only cannot be used together"))
		})
	})

	Describe("code generation", func() {
		DescribeTable("generates correct host and client output",
			func(serviceFile, hostExpectedFile, goClientExpectedFile, pyClientExpectedFile string) {
				serviceCode := readTestdata(serviceFile)
				hostExpected := readTestdata(hostExpectedFile)
				goClientExpected := readTestdata(goClientExpectedFile)
				pyClientExpected := readTestdata(pyClientExpectedFile)

				Expect(os.WriteFile(filepath.Join(testDir, "service.go"), []byte(serviceCode), 0600)).To(Succeed())

				// Generate host and both Go and Python client code
				cmd := exec.Command(hostgenBin, "-input", testDir, "-output", outputDir, "-package", "testpkg", "-go", "-python")
				output, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred(), "Command failed: %s", output)

				// Verify host code
				entries, err := os.ReadDir(outputDir)
				Expect(err).ToNot(HaveOccurred())

				var hostFiles []string
				for _, e := range entries {
					if e.Name() != "go" && e.Name() != "python" && !e.IsDir() {
						hostFiles = append(hostFiles, e.Name())
					}
				}
				Expect(hostFiles).To(HaveLen(1), "Expected exactly one host file, got: %v", hostFiles)

				hostActual, err := os.ReadFile(filepath.Join(outputDir, hostFiles[0]))
				Expect(err).ToNot(HaveOccurred())

				formattedHostActual, err := format.Source(hostActual)
				Expect(err).ToNot(HaveOccurred(), "Generated host code is not valid Go:\n%s", hostActual)

				formattedHostExpected, err := format.Source([]byte(hostExpected))
				Expect(err).ToNot(HaveOccurred(), "Expected host code is not valid Go")

				Expect(string(formattedHostActual)).To(Equal(string(formattedHostExpected)), "Host code mismatch")

				// Verify Go client code
				goDir := filepath.Join(outputDir, "go")
				goClientEntries, err := os.ReadDir(goDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(goClientEntries).To(HaveLen(1), "Expected exactly one Go client file")

				goClientActual, err := os.ReadFile(filepath.Join(goDir, goClientEntries[0].Name()))
				Expect(err).ToNot(HaveOccurred())

				formattedGoClientActual, err := format.Source(goClientActual)
				Expect(err).ToNot(HaveOccurred(), "Generated Go client code is not valid Go:\n%s", goClientActual)

				formattedGoClientExpected, err := format.Source([]byte(goClientExpected))
				Expect(err).ToNot(HaveOccurred(), "Expected Go client code is not valid Go")

				Expect(string(formattedGoClientActual)).To(Equal(string(formattedGoClientExpected)), "Go client code mismatch")

				// Verify Python client code
				pythonDir := filepath.Join(outputDir, "python")
				pyClientEntries, err := os.ReadDir(pythonDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(pyClientEntries).To(HaveLen(1), "Expected exactly one Python client file")

				pyClientActual, err := os.ReadFile(filepath.Join(pythonDir, pyClientEntries[0].Name()))
				Expect(err).ToNot(HaveOccurred())

				Expect(string(pyClientActual)).To(Equal(pyClientExpected), "Python client code mismatch")
			},

			Entry("simple string params",
				"echo_service.go", "echo_expected.go", "echo_client_expected.go", "echo_client_expected.py"),

			Entry("multiple simple params (int32)",
				"math_service.go", "math_expected.go", "math_client_expected.go", "math_client_expected.py"),

			Entry("struct param with request type",
				"store_service.go", "store_expected.go", "store_client_expected.go", "store_client_expected.py"),

			Entry("mixed simple and complex params",
				"list_service.go", "list_expected.go", "list_client_expected.go", "list_client_expected.py"),

			Entry("method without error",
				"counter_service.go", "counter_expected.go", "counter_client_expected.go", "counter_client_expected.py"),

			Entry("no params, error only",
				"ping_service.go", "ping_expected.go", "ping_client_expected.go", "ping_client_expected.py"),

			Entry("map and interface types",
				"meta_service.go", "meta_expected.go", "meta_client_expected.go", "meta_client_expected.py"),

			Entry("pointer types",
				"users_service.go", "users_expected.go", "users_client_expected.go", "users_client_expected.py"),

			Entry("multiple returns",
				"search_service.go", "search_expected.go", "search_client_expected.go", "search_client_expected.py"),

			Entry("bytes",
				"codec_service.go", "codec_expected.go", "codec_client_expected.go", "codec_client_expected.py"),
		)

		It("generates compilable host code for comprehensive service", func() {
			serviceCode := readTestdata("comprehensive_service.go")

			Expect(os.WriteFile(filepath.Join(testDir, "service.go"), []byte(serviceCode), 0600)).To(Succeed())

			// Create go.mod
			goMod := "module testpkg\n\ngo 1.23\n\nrequire github.com/extism/go-sdk v1.7.1\n"
			Expect(os.WriteFile(filepath.Join(testDir, "go.mod"), []byte(goMod), 0600)).To(Succeed())

			// Generate host code only
			cmd := exec.Command(hostgenBin, "-input", testDir, "-output", testDir, "-package", "testpkg", "-host-only")
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "Generation failed: %s", output)

			// Tidy dependencies
			goGetCmd := exec.Command("go", "mod", "tidy")
			goGetCmd.Dir = testDir
			goGetOutput, err := goGetCmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "go mod tidy failed: %s", goGetOutput)

			// Build
			buildCmd := exec.Command("go", "build", ".")
			buildCmd.Dir = testDir
			buildOutput, err := buildCmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "Build failed: %s", buildOutput)
		})

		It("generates compilable client code for comprehensive service", func() {
			serviceCode := readTestdata("comprehensive_service.go")

			Expect(os.WriteFile(filepath.Join(testDir, "service.go"), []byte(serviceCode), 0600)).To(Succeed())

			// Generate client code only to a separate client directory
			clientDir := filepath.Join(outputDir, "client")
			cmd := exec.Command(hostgenBin, "-input", testDir, "-output", clientDir, "-package", "main", "-plugin-only")
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "Generation failed: %s", output)

			// Read generated client code
			goDir := filepath.Join(clientDir, "go")
			entries, err := os.ReadDir(goDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(entries).To(HaveLen(1), "Expected exactly one generated client file")

			content, err := os.ReadFile(filepath.Join(goDir, entries[0].Name()))
			Expect(err).ToNot(HaveOccurred())

			// Verify key expected content first
			contentStr := string(content)
			// Should have wasmimport declarations for all methods
			Expect(contentStr).To(ContainSubstring("//go:wasmimport extism:host/user comprehensive_simpleparams"))
			Expect(contentStr).To(ContainSubstring("//go:wasmimport extism:host/user comprehensive_structparam"))
			Expect(contentStr).To(ContainSubstring("//go:wasmimport extism:host/user comprehensive_noerror"))
			Expect(contentStr).To(ContainSubstring("//go:wasmimport extism:host/user comprehensive_noparams"))
			Expect(contentStr).To(ContainSubstring("//go:wasmimport extism:host/user comprehensive_noparamsnoreturns"))

			// Should have response types for methods with complex returns
			Expect(contentStr).To(ContainSubstring("type ComprehensiveSimpleParamsResponse struct"))
			Expect(contentStr).To(ContainSubstring("type ComprehensiveMultipleReturnsResponse struct"))

			// Should have wrapper functions
			Expect(contentStr).To(ContainSubstring("func ComprehensiveSimpleParams("))
			Expect(contentStr).To(ContainSubstring("func ComprehensiveNoParams()"))
			Expect(contentStr).To(ContainSubstring("func ComprehensiveNoParamsNoReturns()"))

			// Move generated file to clientDir root for compilation
			Expect(os.Rename(filepath.Join(goDir, entries[0].Name()), filepath.Join(clientDir, "nd_host.go"))).To(Succeed())

			// Create go.mod for client code
			goMod := "module main\n\ngo 1.23\n\nrequire github.com/extism/go-pdk v1.1.1\n"
			Expect(os.WriteFile(filepath.Join(clientDir, "go.mod"), []byte(goMod), 0600)).To(Succeed())

			// Add a simple main function for the plugin
			mainGo := `package main

func main() {}
`
			Expect(os.WriteFile(filepath.Join(clientDir, "main.go"), []byte(mainGo), 0600)).To(Succeed())

			// Add type definitions needed by the generated code
			typesGo := `package main

type User2 struct {
	ID   string
	Name string
}

type Filter2 struct {
	Active bool
}
`
			Expect(os.WriteFile(filepath.Join(clientDir, "types.go"), []byte(typesGo), 0600)).To(Succeed())

			// Tidy dependencies
			goTidyCmd := exec.Command("go", "mod", "tidy")
			goTidyCmd.Dir = clientDir
			goTidyOutput, err := goTidyCmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "go mod tidy failed: %s", goTidyOutput)

			// Build as WASM plugin - this validates the client code compiles correctly
			buildCmd := exec.Command("go", "build", "-buildmode=c-shared", "-o", "plugin.wasm", ".")
			buildCmd.Dir = clientDir
			buildCmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
			buildOutput, err := buildCmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "WASM build failed: %s", buildOutput)

			// Verify .wasm file was created
			Expect(filepath.Join(clientDir, "plugin.wasm")).To(BeAnExistingFile())
		})

		It("generates Python client code with -python flag", func() {
			serviceCode := `package testpkg

import "context"

//nd:hostservice name=Test permission=test
type TestService interface {
	//nd:hostfunc
	DoAction(ctx context.Context, input string) (output string, err error)
}
`
			Expect(os.WriteFile(filepath.Join(testDir, "service.go"), []byte(serviceCode), 0600)).To(Succeed())

			cmd := exec.Command(hostgenBin, "-input", testDir, "-output", outputDir, "-package", "testpkg", "-python", "-plugin-only")
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "Command failed: %s", output)

			// Verify Python client code exists
			pythonDir := filepath.Join(outputDir, "python")
			Expect(pythonDir).To(BeADirectory())

			pythonFile := filepath.Join(pythonDir, "nd_host_test.py")
			Expect(pythonFile).To(BeAnExistingFile())

			content, err := os.ReadFile(pythonFile)
			Expect(err).ToNot(HaveOccurred())

			contentStr := string(content)
			Expect(contentStr).To(ContainSubstring("Code generated by hostgen. DO NOT EDIT."))
			Expect(contentStr).To(ContainSubstring("class HostFunctionError(Exception):"))
			Expect(contentStr).To(ContainSubstring(`@extism.import_fn("extism:host/user", "test_doaction")`))
			Expect(contentStr).To(ContainSubstring("def test_do_action(input: str) -> str:"))
		})

		It("generates both Go and Python client code with -go -python flags", func() {
			serviceCode := `package testpkg

import "context"

//nd:hostservice name=Test permission=test
type TestService interface {
	//nd:hostfunc
	DoAction(ctx context.Context, input string) (output string, err error)
}
`
			Expect(os.WriteFile(filepath.Join(testDir, "service.go"), []byte(serviceCode), 0600)).To(Succeed())

			cmd := exec.Command(hostgenBin, "-input", testDir, "-output", outputDir, "-package", "testpkg", "-go", "-python", "-plugin-only")
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "Command failed: %s", output)

			// Verify both Go and Python client code exist
			goDir := filepath.Join(outputDir, "go")
			Expect(goDir).To(BeADirectory())
			Expect(filepath.Join(goDir, "nd_host_test.go")).To(BeAnExistingFile())

			pythonDir := filepath.Join(outputDir, "python")
			Expect(pythonDir).To(BeADirectory())
			Expect(filepath.Join(pythonDir, "nd_host_test.py")).To(BeAnExistingFile())
		})

		It("generates Python code with dataclass for multi-value returns", func() {
			serviceCode := `package testpkg

import "context"

//nd:hostservice name=Cache permission=cache
type CacheService interface {
	//nd:hostfunc
	GetString(ctx context.Context, key string) (value string, exists bool, err error)
}
`
			Expect(os.WriteFile(filepath.Join(testDir, "service.go"), []byte(serviceCode), 0600)).To(Succeed())

			cmd := exec.Command(hostgenBin, "-input", testDir, "-output", outputDir, "-package", "testpkg", "-python", "-plugin-only")
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "Command failed: %s", output)

			content, err := os.ReadFile(filepath.Join(outputDir, "python", "nd_host_cache.py"))
			Expect(err).ToNot(HaveOccurred())

			contentStr := string(content)
			Expect(contentStr).To(ContainSubstring("@dataclass"))
			Expect(contentStr).To(ContainSubstring("class CacheGetStringResult:"))
			Expect(contentStr).To(ContainSubstring("value: str"))
			Expect(contentStr).To(ContainSubstring("exists: bool"))
			Expect(contentStr).To(ContainSubstring("def cache_get_string(key: str) -> CacheGetStringResult:"))
		})

		It("generates Python code for methods with no parameters", func() {
			serviceCode := `package testpkg

import "context"

//nd:hostservice name=Test permission=test
type TestService interface {
	//nd:hostfunc
	Ping(ctx context.Context) (status string, err error)
}
`
			Expect(os.WriteFile(filepath.Join(testDir, "service.go"), []byte(serviceCode), 0600)).To(Succeed())

			cmd := exec.Command(hostgenBin, "-input", testDir, "-output", outputDir, "-package", "testpkg", "-python", "-plugin-only")
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "Command failed: %s", output)

			content, err := os.ReadFile(filepath.Join(outputDir, "python", "nd_host_test.py"))
			Expect(err).ToNot(HaveOccurred())

			contentStr := string(content)
			Expect(contentStr).To(ContainSubstring("def test_ping() -> str:"))
			Expect(contentStr).To(ContainSubstring(`request_bytes = b"{}"`))
		})
	})
})

var testdataDir string

func readTestdata(filename string) string {
	content, err := os.ReadFile(filepath.Join(testdataDir, filename))
	Expect(err).ToNot(HaveOccurred(), "Failed to read testdata file: %s", filename)
	return string(content)
}

func mustGetWd(t FullGinkgoTInterface) string {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root")
		}
		dir = parent
	}
}
