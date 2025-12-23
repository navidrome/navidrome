package main

import (
	"go/format"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var testdataDir string

func readTestdata(filename string) string {
	content, err := os.ReadFile(filepath.Join(testdataDir, filename))
	Expect(err).ToNot(HaveOccurred(), "Failed to read testdata file: %s", filename)
	return string(content)
}

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
	})

	Describe("code generation", func() {
		DescribeTable("generates correct output",
			func(serviceFile, expectedFile string) {
				serviceCode := readTestdata(serviceFile)
				expectedCode := readTestdata(expectedFile)

				Expect(os.WriteFile(filepath.Join(testDir, "service.go"), []byte(serviceCode), 0600)).To(Succeed())

				cmd := exec.Command(hostgenBin, "-input", testDir, "-output", outputDir, "-package", "testpkg")
				output, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred(), "Command failed: %s", output)

				entries, err := os.ReadDir(outputDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(entries).To(HaveLen(1), "Expected exactly one generated file")

				actual, err := os.ReadFile(filepath.Join(outputDir, entries[0].Name()))
				Expect(err).ToNot(HaveOccurred())

				// Format both for comparison
				formattedActual, err := format.Source(actual)
				Expect(err).ToNot(HaveOccurred(), "Generated code is not valid Go:\n%s", actual)

				formattedExpected, err := format.Source([]byte(expectedCode))
				Expect(err).ToNot(HaveOccurred(), "Expected code is not valid Go")

				Expect(string(formattedActual)).To(Equal(string(formattedExpected)))
			},

			Entry("simple string params - no request type needed",
				"echo_service.go", "echo_expected.go"),

			Entry("multiple simple params",
				"math_service.go", "math_expected.go"),

			Entry("struct param with request type",
				"store_service.go", "store_expected.go"),

			Entry("mixed simple and complex params",
				"list_service.go", "list_expected.go"),

			Entry("method without error",
				"counter_service.go", "counter_expected.go"),

			Entry("no params, error only",
				"ping_service.go", "ping_expected.go"),

			Entry("map and interface types",
				"meta_service.go", "meta_expected.go"),

			Entry("pointer types",
				"users_service.go", "users_expected.go"),

			Entry("multiple returns",
				"search_service.go", "search_expected.go"),

			Entry("bytes",
				"codec_service.go", "codec_expected.go"),
		)

		It("generates compilable code for comprehensive service", func() {
			serviceCode := readTestdata("comprehensive_service.go")

			Expect(os.WriteFile(filepath.Join(testDir, "service.go"), []byte(serviceCode), 0600)).To(Succeed())

			// Create go.mod
			goMod := "module testpkg\n\ngo 1.23\n\nrequire github.com/extism/go-sdk v1.7.1\n"
			Expect(os.WriteFile(filepath.Join(testDir, "go.mod"), []byte(goMod), 0600)).To(Succeed())

			// Generate
			cmd := exec.Command(hostgenBin, "-input", testDir, "-output", testDir, "-package", "testpkg")
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
	})
})

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
