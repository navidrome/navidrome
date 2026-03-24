package main

import (
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// normalizeGeneratedCode normalizes generated code for comparison with expected output.
func normalizeGeneratedCode(code string) string {
	// Replace package names (generated uses ndpdk, testdata may use ndhost)
	code = strings.ReplaceAll(code, "package ndhost", "package ndpdk")
	return code
}

var _ = Describe("ndpgen CLI", Ordered, func() {
	var (
		testDir   string
		outputDir string
		ndpgenBin string
	)

	BeforeAll(func() {
		// Set testdata directory (relative to ndpgen root)
		testdataDir = filepath.Join(mustGetWd(GinkgoT()), "testdata")

		// Build the ndpgen binary
		ndpgenBin = filepath.Join(os.TempDir(), "ndpgen-test")
		cmd := exec.Command("go", "build", "-o", ndpgenBin, ".")
		cmd.Dir = mustGetWd(GinkgoT())
		output, err := cmd.CombinedOutput()
		Expect(err).ToNot(HaveOccurred(), "Failed to build ndpgen: %s", output)
		DeferCleanup(func() {
			os.Remove(ndpgenBin)
		})
	})

	BeforeEach(func() {
		var err error
		testDir, err = os.MkdirTemp("", "ndpgen-test-input-*")
		Expect(err).ToNot(HaveOccurred())
		outputDir, err = os.MkdirTemp("", "ndpgen-test-output-*")
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
			cmd := exec.Command(ndpgenBin, "-input", testDir, "-output", outputDir, "-package", "ndpdk", "-v")
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "Command failed: %s", output)

			outputStr := string(output)
			Expect(outputStr).To(ContainSubstring("Input directory:"))
			Expect(outputStr).To(ContainSubstring("Base output directory:"))
			Expect(outputStr).To(ContainSubstring("Go output directory:"))
			Expect(outputStr).To(ContainSubstring("Found 1 host service(s)"))
			Expect(outputStr).To(ContainSubstring("Generated"))
		})

		It("supports dry-run mode", func() {
			cmd := exec.Command(ndpgenBin, "-input", testDir, "-output", outputDir, "-package", "ndpdk", "-dry-run")
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "Command failed: %s", output)

			Expect(string(output)).To(ContainSubstring("func TestDoAction("))
			Expect(filepath.Join(outputDir, "nd_host_test.go")).ToNot(BeAnExistingFile())
		})

		It("uses default package name 'host'", func() {
			customOutput, err := os.MkdirTemp("", "mypkg")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(customOutput)

			cmd := exec.Command(ndpgenBin, "-input", testDir, "-output", customOutput)
			_, err = cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())

			// Go code goes to $output/go/host/
			content, err := os.ReadFile(filepath.Join(customOutput, "go", "host", "nd_host_test.go"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("package host"))
		})

		It("returns error for invalid input directory", func() {
			cmd := exec.Command(ndpgenBin, "-input", "/nonexistent/path")
			output, err := cmd.CombinedOutput()
			Expect(err).To(HaveOccurred())
			Expect(string(output)).To(ContainSubstring("parsing source files"))
		})

		It("handles no annotated services gracefully", func() {
			Expect(os.WriteFile(filepath.Join(testDir, "service.go"), []byte("package testpkg\n"), 0600)).To(Succeed())

			cmd := exec.Command(ndpgenBin, "-input", testDir, "-output", outputDir, "-v")
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

			cmd := exec.Command(ndpgenBin, "-input", testDir, "-output", outputDir, "-package", "ndpdk", "-v")
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "Command failed: %s", output)
			Expect(string(output)).To(ContainSubstring("Found 2 host service(s)"))

			// Go code goes to $output/go/host/
			goHostDir := filepath.Join(outputDir, "go", "host")
			Expect(filepath.Join(goHostDir, "nd_host_servicea.go")).To(BeAnExistingFile())
			Expect(filepath.Join(goHostDir, "nd_host_serviceb.go")).To(BeAnExistingFile())
		})

		It("generates Go client code by default", func() {
			cmd := exec.Command(ndpgenBin, "-input", testDir, "-output", outputDir, "-package", "ndpdk")
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "Command failed: %s", output)

			// Go client code goes to $output/go/host/
			goHostDir := filepath.Join(outputDir, "go", "host")
			Expect(filepath.Join(goHostDir, "nd_host_test.go")).To(BeAnExistingFile())
			// Stub file also generated
			Expect(filepath.Join(goHostDir, "nd_host_test_stub.go")).To(BeAnExistingFile())
			// doc.go in host dir
			Expect(filepath.Join(goHostDir, "doc.go")).To(BeAnExistingFile())
			// go.mod at parent $output/go/ for consolidated module
			goDir := filepath.Join(outputDir, "go")
			Expect(filepath.Join(goDir, "go.mod")).To(BeAnExistingFile())
		})
	})

	Describe("code generation", func() {
		DescribeTable("generates correct client output",
			func(serviceFile, goClientExpectedFile, pyClientExpectedFile, rsClientExpectedFile string) {
				serviceCode := readTestdata(serviceFile)
				goClientExpected := readTestdata(goClientExpectedFile)
				pyClientExpected := readTestdata(pyClientExpectedFile)
				rsClientExpected := readTestdata(rsClientExpectedFile)

				Expect(os.WriteFile(filepath.Join(testDir, "service.go"), []byte(serviceCode), 0600)).To(Succeed())

				// Generate all client code (Go, Python, Rust)
				cmd := exec.Command(ndpgenBin, "-input", testDir, "-output", outputDir, "-package", "ndpdk", "-go", "-python", "-rust")
				output, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred(), "Command failed: %s", output)

				// Verify Go client code (now in $output/go/host/)
				goHostDir := filepath.Join(outputDir, "go", "host")
				entries, err := os.ReadDir(goHostDir)
				Expect(err).ToNot(HaveOccurred())

				var goClientFiles []string
				for _, e := range entries {
					if !e.IsDir() &&
						!strings.HasSuffix(e.Name(), "_stub.go") &&
						e.Name() != "doc.go" && e.Name() != "go.mod" {
						goClientFiles = append(goClientFiles, e.Name())
					}
				}
				Expect(goClientFiles).To(HaveLen(1), "Expected exactly one Go client file, got: %v", goClientFiles)

				goClientActual, err := os.ReadFile(filepath.Join(goHostDir, goClientFiles[0]))
				Expect(err).ToNot(HaveOccurred())

				formattedGoClientActual, err := format.Source(goClientActual)
				Expect(err).ToNot(HaveOccurred(), "Generated Go client code is not valid Go:\n%s", goClientActual)

				// Normalize expected code to match ndpgen output format
				normalizedExpected := normalizeGeneratedCode(goClientExpected)
				formattedGoClientExpected, err := format.Source([]byte(normalizedExpected))
				Expect(err).ToNot(HaveOccurred(), "Expected Go client code is not valid Go")

				Expect(string(formattedGoClientActual)).To(Equal(string(formattedGoClientExpected)), "Go client code mismatch")

				// Verify Python client code (now in $output/python/host/)
				pythonHostDir := filepath.Join(outputDir, "python", "host")
				pyClientEntries, err := os.ReadDir(pythonHostDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(pyClientEntries).To(HaveLen(1), "Expected exactly one Python client file")

				pyClientActual, err := os.ReadFile(filepath.Join(pythonHostDir, pyClientEntries[0].Name()))
				Expect(err).ToNot(HaveOccurred())

				Expect(string(pyClientActual)).To(Equal(pyClientExpected), "Python client code mismatch")

				// Verify Rust client code (now in $output/rust/nd-pdk-host/src/)
				rustSrcDir := filepath.Join(outputDir, "rust", "nd-pdk-host", "src")
				rsClientEntries, err := os.ReadDir(rustSrcDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(rsClientEntries).To(HaveLen(2), "Expected Rust client file and lib.rs in src/")

				// Find the client file (not lib.rs)
				var rsClientName string
				for _, entry := range rsClientEntries {
					if entry.Name() != "lib.rs" {
						rsClientName = entry.Name()
						break
					}
				}
				Expect(rsClientName).ToNot(BeEmpty(), "Expected to find Rust client file")

				rsClientActual, err := os.ReadFile(filepath.Join(rustSrcDir, rsClientName))
				Expect(err).ToNot(HaveOccurred())

				Expect(string(rsClientActual)).To(Equal(rsClientExpected), "Rust client code mismatch")
			},

			Entry("simple string params",
				"echo_service.go.txt", "echo_client_expected.go.txt", "echo_client_expected.py", "echo_client_expected.rs"),

			Entry("multiple simple params (int32)",
				"math_service.go.txt", "math_client_expected.go.txt", "math_client_expected.py", "math_client_expected.rs"),

			Entry("struct param with request type",
				"store_service.go.txt", "store_client_expected.go.txt", "store_client_expected.py", "store_client_expected.rs"),

			Entry("mixed simple and complex params",
				"list_service.go.txt", "list_client_expected.go.txt", "list_client_expected.py", "list_client_expected.rs"),

			Entry("method without error",
				"counter_service.go.txt", "counter_client_expected.go.txt", "counter_client_expected.py", "counter_client_expected.rs"),

			Entry("no params, error only",
				"ping_service.go.txt", "ping_client_expected.go.txt", "ping_client_expected.py", "ping_client_expected.rs"),

			Entry("map and interface types",
				"meta_service.go.txt", "meta_client_expected.go.txt", "meta_client_expected.py", "meta_client_expected.rs"),

			Entry("pointer types",
				"users_service.go.txt", "users_client_expected.go.txt", "users_client_expected.py", "users_client_expected.rs"),

			Entry("multiple returns",
				"search_service.go.txt", "search_client_expected.go.txt", "search_client_expected.py", "search_client_expected.rs"),

			Entry("bytes",
				"codec_service.go.txt", "codec_client_expected.go.txt", "codec_client_expected.py", "codec_client_expected.rs"),

			Entry("option pattern (value, exists bool)",
				"config_service.go.txt", "config_client_expected.go.txt", "config_client_expected.py", "config_client_expected.rs"),
		)

		It("generates compilable client code for comprehensive service", func() {
			serviceCode := readTestdata("comprehensive_service.go.txt")

			Expect(os.WriteFile(filepath.Join(testDir, "service.go"), []byte(serviceCode), 0600)).To(Succeed())

			// Generate client code
			cmd := exec.Command(ndpgenBin, "-input", testDir, "-output", outputDir, "-package", "ndpdk")
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "Generation failed: %s", output)

			// Go code goes to $output/go/host/
			goHostDir := filepath.Join(outputDir, "go", "host")

			// Read generated client code
			entries, err := os.ReadDir(goHostDir)
			Expect(err).ToNot(HaveOccurred())

			// Find the client file
			var clientFileName string
			for _, entry := range entries {
				name := entry.Name()
				if name != "doc.go" && name != "go.mod" && !strings.HasSuffix(name, "_stub.go") && strings.HasSuffix(name, ".go") {
					clientFileName = name
					break
				}
			}
			Expect(clientFileName).ToNot(BeEmpty(), "Expected to find Go client file")

			content, err := os.ReadFile(filepath.Join(goHostDir, clientFileName))
			Expect(err).ToNot(HaveOccurred())

			// Verify key expected content
			contentStr := string(content)
			// Should have wasmimport declarations for all methods
			Expect(contentStr).To(ContainSubstring("//go:wasmimport extism:host/user comprehensive_simpleparams"))
			Expect(contentStr).To(ContainSubstring("//go:wasmimport extism:host/user comprehensive_structparam"))
			Expect(contentStr).To(ContainSubstring("//go:wasmimport extism:host/user comprehensive_noerror"))
			Expect(contentStr).To(ContainSubstring("//go:wasmimport extism:host/user comprehensive_noparams"))
			Expect(contentStr).To(ContainSubstring("//go:wasmimport extism:host/user comprehensive_noparamsnoreturns"))

			// Should have response types for methods with complex returns (private types in client code)
			Expect(contentStr).To(ContainSubstring("type comprehensiveSimpleParamsResponse struct"))
			Expect(contentStr).To(ContainSubstring("type comprehensiveMultipleReturnsResponse struct"))

			// Should have wrapper functions
			Expect(contentStr).To(ContainSubstring("func ComprehensiveSimpleParams("))
			Expect(contentStr).To(ContainSubstring("func ComprehensiveNoParams()"))
			Expect(contentStr).To(ContainSubstring("func ComprehensiveNoParamsNoReturns()"))

			// Create a plugin directory with proper import structure
			pluginDir := filepath.Join(outputDir, "plugin")
			Expect(os.MkdirAll(pluginDir, 0750)).To(Succeed())

			// go.mod is at parent $output/go/ for consolidated module
			goDir := filepath.Join(outputDir, "go")

			// Create go.mod for the plugin that imports the generated library
			goMod := fmt.Sprintf(`module testplugin

go 1.25

require github.com/navidrome/navidrome/plugins/pdk/go v0.0.0

replace github.com/navidrome/navidrome/plugins/pdk/go => %s
`, goDir)
			Expect(os.WriteFile(filepath.Join(pluginDir, "go.mod"), []byte(goMod), 0600)).To(Succeed())

			// Add a simple main function that imports and uses the ndpdk package
			mainGo := `package main

import ndpdk "github.com/navidrome/navidrome/plugins/pdk/go/host"

func main() {}

// Use some functions to ensure import is not unused
var _ = ndpdk.ComprehensiveNoParams
`
			Expect(os.WriteFile(filepath.Join(pluginDir, "main.go"), []byte(mainGo), 0600)).To(Succeed())

			// Tidy dependencies for the generated go library
			goTidyLibCmd := exec.Command("go", "mod", "tidy")
			goTidyLibCmd.Dir = goDir
			goTidyLibOutput, err := goTidyLibCmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "go mod tidy (library) failed: %s", goTidyLibOutput)

			// Tidy dependencies for the plugin
			goTidyCmd := exec.Command("go", "mod", "tidy")
			goTidyCmd.Dir = pluginDir
			goTidyOutput, err := goTidyCmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "go mod tidy (plugin) failed: %s", goTidyOutput)

			// Build as WASM plugin - this validates the client code compiles correctly
			buildCmd := exec.Command("go", "build", "-buildmode=c-shared", "-o", "plugin.wasm", ".")
			buildCmd.Dir = pluginDir
			buildCmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
			buildOutput, err := buildCmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "WASM build failed: %s", buildOutput)

			// Verify .wasm file was created
			Expect(filepath.Join(pluginDir, "plugin.wasm")).To(BeAnExistingFile())
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

			cmd := exec.Command(ndpgenBin, "-input", testDir, "-output", outputDir, "-package", "ndpdk", "-python")
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "Command failed: %s", output)

			// Verify Python client code exists in $output/python/host/
			pythonHostDir := filepath.Join(outputDir, "python", "host")
			Expect(pythonHostDir).To(BeADirectory())

			pythonFile := filepath.Join(pythonHostDir, "nd_host_test.py")
			Expect(pythonFile).To(BeAnExistingFile())

			content, err := os.ReadFile(pythonFile)
			Expect(err).ToNot(HaveOccurred())

			contentStr := string(content)
			Expect(contentStr).To(ContainSubstring("Code generated by ndpgen. DO NOT EDIT."))
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

			cmd := exec.Command(ndpgenBin, "-input", testDir, "-output", outputDir, "-package", "ndpdk", "-go", "-python")
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "Command failed: %s", output)

			// Verify Go client code exists in $output/go/host/
			goHostDir := filepath.Join(outputDir, "go", "host")
			Expect(filepath.Join(goHostDir, "nd_host_test.go")).To(BeAnExistingFile())

			// Verify Python client code exists in $output/python/host/
			pythonHostDir := filepath.Join(outputDir, "python", "host")
			Expect(pythonHostDir).To(BeADirectory())
			Expect(filepath.Join(pythonHostDir, "nd_host_test.py")).To(BeAnExistingFile())
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

			cmd := exec.Command(ndpgenBin, "-input", testDir, "-output", outputDir, "-package", "ndpdk", "-python")
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "Command failed: %s", output)

			content, err := os.ReadFile(filepath.Join(outputDir, "python", "host", "nd_host_cache.py"))
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

			cmd := exec.Command(ndpgenBin, "-input", testDir, "-output", outputDir, "-package", "ndpdk", "-python")
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "Command failed: %s", output)

			content, err := os.ReadFile(filepath.Join(outputDir, "python", "host", "nd_host_test.py"))
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
	// Look for ndpgen's own go.mod (the subproject root)
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			// Check if this is the ndpgen go.mod by reading it
			content, err := os.ReadFile(goModPath)
			if err == nil && strings.Contains(string(content), "plugins/cmd/ndpgen") {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find ndpgen project root")
		}
		dir = parent
	}
}
