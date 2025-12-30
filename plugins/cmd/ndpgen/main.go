// ndpgen generates Navidrome Plugin Development Kit (PDK) code from annotated Go interfaces.
//
// This is the unified code generator that handles both host function wrappers
// and capability export wrappers.
//
// Usage:
//
//	# Generate host wrappers (from plugins/host to plugins/pdk)
//	ndpgen -host-only -input=./plugins/host -output=./plugins/pdk
//
//	# Generate capability wrappers (from plugins/capabilities to plugins/pdk)
//	ndpgen -capability-only -input=./plugins/capabilities -output=./plugins/pdk
//
//	# Generate XTP schemas from capabilities (output to input directory)
//	ndpgen -schemas -input=./plugins/capabilities
//
// Output directories:
//   - Host functions:  $output/go/host/, $output/python/host/, $output/rust/host/
//   - Capabilities:    $output/go/<capability>/ (e.g., $output/go/metadata/)
//   - Schemas:         $input/<capability>.yaml (co-located with Go sources)
//
// Flags:
//
//	-input           Input directory containing Go source files with annotated interfaces
//	-output          Output directory base for generated files (default: same as input)
//	-package         Output package name for Go (default: host for host-only, auto for capabilities)
//	-host-only       Generate only host function wrappers
//	-capability-only Generate only capability export wrappers
//	-schemas         Generate XTP YAML schemas from capabilities
//	-go              Generate Go client wrappers (default: true when not using -python/-rust)
//	-python          Generate Python client wrappers (default: false)
//	-rust            Generate Rust client wrappers (default: false)
//	-v               Verbose output
//	-dry-run         Preview generated code without writing files
package main

import (
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/plugins/cmd/ndpgen/internal"
)

// config holds the parsed command-line configuration.
type config struct {
	inputDir         string
	outputDir        string // Base output directory (e.g., plugins/pdk)
	goOutputDir      string // Go output: $outputDir/go/host (for host-only)
	pythonOutputDir  string // Python output: $outputDir/python/host
	rustOutputDir    string // Rust output: $outputDir/rust/host
	pkgName          string
	hostOnly         bool
	capabilityOnly   bool
	schemasOnly      bool // Generate XTP schemas from capabilities (output goes to inputDir)
	generateGoClient bool
	generatePyClient bool
	generateRsClient bool
	verbose          bool
	dryRun           bool
}

func main() {
	cfg, err := parseConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if cfg.schemasOnly {
		if err := runSchemaGeneration(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if cfg.capabilityOnly {
		if err := runCapabilityGeneration(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Default: host-only mode
	services, err := parseServices(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if len(services) == 0 {
		return
	}

	if err := generateAllCode(cfg, services); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runCapabilityGeneration handles capability-only code generation.
func runCapabilityGeneration(cfg *config) error {
	capabilities, err := parseCapabilities(cfg)
	if err != nil {
		return err
	}
	if len(capabilities) == 0 {
		if cfg.verbose {
			fmt.Println("No capabilities found")
		}
		return nil
	}

	return generateCapabilityCode(cfg, capabilities)
}

// runSchemaGeneration handles XTP schema generation from capabilities.
func runSchemaGeneration(cfg *config) error {
	capabilities, err := parseCapabilities(cfg)
	if err != nil {
		return err
	}
	if len(capabilities) == 0 {
		if cfg.verbose {
			fmt.Println("No capabilities found")
		}
		return nil
	}

	return generateSchemas(cfg, capabilities)
}

// parseConfig parses command-line flags and returns the configuration.
func parseConfig() (*config, error) {
	var (
		inputDir       = flag.String("input", ".", "Input directory containing Go source files")
		outputDir      = flag.String("output", "", "Base output directory for generated files (default: same as input)")
		pkgName        = flag.String("package", "", "Output package name for Go (default: host for host-only, auto for capabilities)")
		hostOnly       = flag.Bool("host-only", false, "Generate only host function wrappers")
		capabilityOnly = flag.Bool("capability-only", false, "Generate only capability export wrappers")
		schemasOnly    = flag.Bool("schemas", false, "Generate XTP YAML schemas from capabilities (output to input directory)")
		goClient       = flag.Bool("go", false, "Generate Go client wrappers")
		pyClient       = flag.Bool("python", false, "Generate Python client wrappers")
		rsClient       = flag.Bool("rust", false, "Generate Rust client wrappers")
		verbose        = flag.Bool("v", false, "Verbose output")
		dryRun         = flag.Bool("dry-run", false, "Preview generated code without writing files")
	)
	flag.Parse()

	// Count how many mode flags are specified
	modeCount := 0
	if *hostOnly {
		modeCount++
	}
	if *capabilityOnly {
		modeCount++
	}
	if *schemasOnly {
		modeCount++
	}

	// Default to host-only if no mode is specified
	if modeCount == 0 {
		*hostOnly = true
	}

	// Cannot specify multiple modes
	if modeCount > 1 {
		return nil, fmt.Errorf("cannot specify multiple modes (-host-only, -capability-only, -schemas)")
	}

	if *outputDir == "" {
		*outputDir = *inputDir
	}

	// Default package name based on mode
	if *pkgName == "" {
		if *hostOnly {
			*pkgName = "host"
		}
		// For capability-only, package name is derived from capability annotation
	}

	absInput, err := filepath.Abs(*inputDir)
	if err != nil {
		return nil, fmt.Errorf("resolving input path: %w", err)
	}
	absOutput, err := filepath.Abs(*outputDir)
	if err != nil {
		return nil, fmt.Errorf("resolving output path: %w", err)
	}

	// Set output directories for each language: $output/$lang/host/
	absGoOutput := filepath.Join(absOutput, "go", "host")
	absPythonOutput := filepath.Join(absOutput, "python", "host")
	absRustOutput := filepath.Join(absOutput, "rust", "host")

	// Determine what to generate
	// Default: generate Go clients if no language flag is specified
	anyLangFlag := *goClient || *pyClient || *rsClient

	return &config{
		inputDir:         absInput,
		outputDir:        absOutput,
		goOutputDir:      absGoOutput,
		pythonOutputDir:  absPythonOutput,
		rustOutputDir:    absRustOutput,
		pkgName:          *pkgName,
		hostOnly:         *hostOnly,
		capabilityOnly:   *capabilityOnly,
		schemasOnly:      *schemasOnly,
		generateGoClient: *goClient || !anyLangFlag,
		generatePyClient: *pyClient,
		generateRsClient: *rsClient,
		verbose:          *verbose,
		dryRun:           *dryRun,
	}, nil
}

// parseServices parses source files and returns discovered services.
func parseServices(cfg *config) ([]internal.Service, error) {
	if cfg.verbose {
		fmt.Printf("Input directory: %s\n", cfg.inputDir)
		fmt.Printf("Base output directory: %s\n", cfg.outputDir)
		if cfg.generateGoClient {
			fmt.Printf("Go output directory: %s\n", cfg.goOutputDir)
		}
		if cfg.generatePyClient {
			fmt.Printf("Python output directory: %s\n", cfg.pythonOutputDir)
		}
		if cfg.generateRsClient {
			fmt.Printf("Rust output directory: %s\n", cfg.rustOutputDir)
		}
		fmt.Printf("Package name: %s\n", cfg.pkgName)
		fmt.Printf("Host-only mode: %v\n", cfg.hostOnly)
		fmt.Printf("Generate Go client code: %v\n", cfg.generateGoClient)
		fmt.Printf("Generate Python client code: %v\n", cfg.generatePyClient)
		fmt.Printf("Generate Rust client code: %v\n", cfg.generateRsClient)
	}

	services, err := internal.ParseDirectory(cfg.inputDir)
	if err != nil {
		return nil, fmt.Errorf("parsing source files: %w", err)
	}

	if len(services) == 0 {
		if cfg.verbose {
			fmt.Println("No host services found")
		}
		return nil, nil
	}

	if cfg.verbose {
		fmt.Printf("Found %d host service(s)\n", len(services))
		for _, svc := range services {
			fmt.Printf("  - %s (%d methods)\n", svc.Name, len(svc.Methods))
		}
	}

	return services, nil
}

// parseCapabilities parses source files and returns discovered capabilities.
func parseCapabilities(cfg *config) ([]internal.Capability, error) {
	if cfg.verbose {
		fmt.Printf("Input directory: %s\n", cfg.inputDir)
		fmt.Printf("Base output directory: %s\n", cfg.outputDir)
		fmt.Printf("Capability-only mode: %v\n", cfg.capabilityOnly)
	}

	capabilities, err := internal.ParseCapabilities(cfg.inputDir)
	if err != nil {
		return nil, fmt.Errorf("parsing capability files: %w", err)
	}

	if len(capabilities) == 0 {
		return nil, nil
	}

	if cfg.verbose {
		fmt.Printf("Found %d capability(ies)\n", len(capabilities))
		for _, cap := range capabilities {
			fmt.Printf("  - %s (%d exports, required=%v)\n", cap.Name, len(cap.Methods), cap.Required)
		}
	}

	return capabilities, nil
}

// generateCapabilityCode generates Go export wrappers for all capabilities.
func generateCapabilityCode(cfg *config, capabilities []internal.Capability) error {
	for _, cap := range capabilities {
		// Output directory is $output/go/<capability_name>/
		outputDir := filepath.Join(cfg.outputDir, "go", cap.Name)

		if err := generateCapabilityGoCode(cap, outputDir, cfg.dryRun, cfg.verbose); err != nil {
			return fmt.Errorf("generating capability code for %s: %w", cap.Name, err)
		}
	}

	return nil
}

// generateCapabilityGoCode generates Go export wrapper code for a capability.
func generateCapabilityGoCode(cap internal.Capability, outputDir string, dryRun, verbose bool) error {
	// Use the capability name as the package name
	pkgName := cap.Name

	// Generate the main WASM code
	code, err := internal.GenerateCapabilityGo(cap, pkgName)
	if err != nil {
		return fmt.Errorf("generating code: %w", err)
	}

	formatted, err := format.Source(code)
	if err != nil {
		return fmt.Errorf("formatting code: %w\nRaw code:\n%s", err, code)
	}

	mainFile := filepath.Join(outputDir, cap.Name+".go")

	if dryRun {
		fmt.Printf("=== %s ===\n%s\n", mainFile, formatted)
	} else {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}

		if err := os.WriteFile(mainFile, formatted, 0600); err != nil {
			return fmt.Errorf("writing file: %w", err)
		}

		if verbose {
			fmt.Printf("Generated capability code: %s\n", mainFile)
		}
	}

	// Generate the stub code for non-WASM platforms
	stubCode, err := internal.GenerateCapabilityGoStub(cap, pkgName)
	if err != nil {
		return fmt.Errorf("generating stub code: %w", err)
	}

	formattedStub, err := format.Source(stubCode)
	if err != nil {
		return fmt.Errorf("formatting stub code: %w\nRaw code:\n%s", err, stubCode)
	}

	stubFile := filepath.Join(outputDir, cap.Name+"_stub.go")

	if dryRun {
		fmt.Printf("=== %s ===\n%s\n", stubFile, formattedStub)
	} else {
		if err := os.WriteFile(stubFile, formattedStub, 0600); err != nil {
			return fmt.Errorf("writing stub file: %w", err)
		}

		if verbose {
			fmt.Printf("Generated capability stub: %s\n", stubFile)
		}
	}

	return nil
}

// generateAllCode generates all requested code for the services.
func generateAllCode(cfg *config, services []internal.Service) error {
	for _, svc := range services {
		if cfg.generateGoClient {
			if err := generateGoClientCode(svc, cfg.goOutputDir, cfg.pkgName, cfg.dryRun, cfg.verbose); err != nil {
				return fmt.Errorf("generating Go client code for %s: %w", svc.Name, err)
			}
		}
		if cfg.generatePyClient {
			if err := generatePythonClientCode(svc, cfg.pythonOutputDir, cfg.dryRun, cfg.verbose); err != nil {
				return fmt.Errorf("generating Python client code for %s: %w", svc.Name, err)
			}
		}
		if cfg.generateRsClient {
			if err := generateRustClientCode(svc, cfg.rustOutputDir, cfg.dryRun, cfg.verbose); err != nil {
				return fmt.Errorf("generating Rust client code for %s: %w", svc.Name, err)
			}
		}
	}

	if cfg.generateRsClient && len(services) > 0 {
		if err := generateRustLibFile(services, cfg.rustOutputDir, cfg.dryRun, cfg.verbose); err != nil {
			return fmt.Errorf("generating Rust lib.rs: %w", err)
		}
	}

	if cfg.generateGoClient && len(services) > 0 {
		if err := generateGoDocFile(services, cfg.goOutputDir, cfg.pkgName, cfg.dryRun, cfg.verbose); err != nil {
			return fmt.Errorf("generating Go doc.go: %w", err)
		}
		if err := generateGoModFile(cfg.goOutputDir, cfg.dryRun, cfg.verbose); err != nil {
			return fmt.Errorf("generating Go go.mod: %w", err)
		}
	}

	return nil
}

// generateGoClientCode generates Go client-side code for a service.
func generateGoClientCode(svc internal.Service, outputDir, pkgName string, dryRun, verbose bool) error {
	code, err := internal.GenerateClientGo(svc, pkgName)
	if err != nil {
		return fmt.Errorf("generating code: %w", err)
	}

	formatted, err := format.Source(code)
	if err != nil {
		return fmt.Errorf("formatting code: %w\nRaw code:\n%s", err, code)
	}

	// Client code goes directly in the output directory
	clientFile := filepath.Join(outputDir, "nd_host_"+strings.ToLower(svc.Name)+".go")

	if dryRun {
		fmt.Printf("=== %s ===\n%s\n", clientFile, formatted)
	} else {
		// Create output directory if needed
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}

		if err := os.WriteFile(clientFile, formatted, 0600); err != nil {
			return fmt.Errorf("writing file: %w", err)
		}

		if verbose {
			fmt.Printf("Generated Go client code: %s\n", clientFile)
		}
	}

	// Also generate stub file for non-WASM platforms
	return generateGoClientStubCode(svc, outputDir, pkgName, dryRun, verbose)
}

// generateGoClientStubCode generates stub code for non-WASM platforms.
func generateGoClientStubCode(svc internal.Service, outputDir, pkgName string, dryRun, verbose bool) error {
	code, err := internal.GenerateClientGoStub(svc, pkgName)
	if err != nil {
		return fmt.Errorf("generating stub code: %w", err)
	}

	formatted, err := format.Source(code)
	if err != nil {
		return fmt.Errorf("formatting stub code: %w\nRaw code:\n%s", err, code)
	}

	// Stub code goes directly in output directory with _stub suffix
	stubFile := filepath.Join(outputDir, "nd_host_"+strings.ToLower(svc.Name)+"_stub.go")

	if dryRun {
		fmt.Printf("=== %s ===\n%s\n", stubFile, formatted)
		return nil
	}

	// Create output directory if needed
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	if err := os.WriteFile(stubFile, formatted, 0600); err != nil {
		return fmt.Errorf("writing stub file: %w", err)
	}

	if verbose {
		fmt.Printf("Generated Go client stub: %s\n", stubFile)
	}
	return nil
}

// generatePythonClientCode generates Python client-side code for a service.
func generatePythonClientCode(svc internal.Service, outputDir string, dryRun, verbose bool) error {
	code, err := internal.GenerateClientPython(svc)
	if err != nil {
		return fmt.Errorf("generating code: %w", err)
	}

	// Python code goes directly in the output directory
	clientFile := filepath.Join(outputDir, "nd_host_"+strings.ToLower(svc.Name)+".py")

	if dryRun {
		fmt.Printf("=== %s ===\n%s\n", clientFile, code)
		return nil
	}

	// Create output directory if needed
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating python client directory: %w", err)
	}

	if err := os.WriteFile(clientFile, code, 0600); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	if verbose {
		fmt.Printf("Generated Python client code: %s\n", clientFile)
	}
	return nil
}

// generateRustClientCode generates Rust client-side code for a service.
func generateRustClientCode(svc internal.Service, outputDir string, dryRun, verbose bool) error {
	code, err := internal.GenerateClientRust(svc)
	if err != nil {
		return fmt.Errorf("generating code: %w", err)
	}

	// Rust code goes directly in the output directory
	clientFile := filepath.Join(outputDir, "nd_host_"+strings.ToLower(svc.Name)+".rs")

	if dryRun {
		fmt.Printf("=== %s ===\n%s\n", clientFile, code)
		return nil
	}

	// Create output directory if needed
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating rust client directory: %w", err)
	}

	if err := os.WriteFile(clientFile, code, 0600); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	if verbose {
		fmt.Printf("Generated Rust client code: %s\n", clientFile)
	}
	return nil
}

// generateRustLibFile generates the lib.rs file that exposes all Rust modules.
func generateRustLibFile(services []internal.Service, outputDir string, dryRun, verbose bool) error {
	code, err := internal.GenerateRustLib(services)
	if err != nil {
		return fmt.Errorf("generating lib.rs: %w", err)
	}

	// lib.rs goes directly in the output directory
	libFile := filepath.Join(outputDir, "lib.rs")

	if dryRun {
		fmt.Printf("=== %s ===\n%s\n", libFile, code)
		return nil
	}

	// Create output directory if needed
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating rust client directory: %w", err)
	}

	if err := os.WriteFile(libFile, code, 0600); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	if verbose {
		fmt.Printf("Generated Rust lib.rs: %s\n", libFile)
	}
	return nil
}

// generateGoDocFile generates the doc.go file for the Go library.
func generateGoDocFile(services []internal.Service, outputDir, pkgName string, dryRun, verbose bool) error {
	code, err := internal.GenerateGoDoc(services, pkgName)
	if err != nil {
		return fmt.Errorf("generating doc.go: %w", err)
	}

	formatted, err := format.Source(code)
	if err != nil {
		return fmt.Errorf("formatting doc.go: %w\nRaw code:\n%s", err, code)
	}

	docFile := filepath.Join(outputDir, "doc.go")

	if dryRun {
		fmt.Printf("=== %s ===\n%s\n", docFile, formatted)
		return nil
	}

	// Create output directory if needed
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	if err := os.WriteFile(docFile, formatted, 0600); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	if verbose {
		fmt.Printf("Generated Go doc.go: %s\n", docFile)
	}
	return nil
}

// generateGoModFile generates the go.mod file for the Go library.
// The go.mod is placed at the parent directory ($output/go/) to create a unified
// module that includes both host wrappers and capabilities.
func generateGoModFile(outputDir string, dryRun, verbose bool) error {
	code, err := internal.GenerateGoMod()
	if err != nil {
		return fmt.Errorf("generating go.mod: %w", err)
	}

	// Output to parent directory ($output/go/) instead of host directory
	parentDir := filepath.Dir(outputDir)
	modFile := filepath.Join(parentDir, "go.mod")

	if dryRun {
		fmt.Printf("=== %s ===\n%s\n", modFile, code)
		return nil
	}

	// Create parent directory if needed
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	if err := os.WriteFile(modFile, code, 0600); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	if verbose {
		fmt.Printf("Generated Go go.mod: %s\n", modFile)
	}
	return nil
}

// generateSchemas generates XTP YAML schemas from capabilities.
func generateSchemas(cfg *config, capabilities []internal.Capability) error {
	for _, cap := range capabilities {
		if err := generateSchemaFile(cap, cfg.inputDir, cfg.dryRun, cfg.verbose); err != nil {
			return fmt.Errorf("generating schema for %s: %w", cap.Name, err)
		}
	}
	return nil
}

// generateSchemaFile generates an XTP YAML schema file for a capability.
func generateSchemaFile(cap internal.Capability, outputDir string, dryRun, verbose bool) error {
	schema, err := internal.GenerateSchema(cap)
	if err != nil {
		return fmt.Errorf("generating schema: %w", err)
	}

	// Use the source file name: websocket_callback.go -> websocket_callback.yaml
	schemaFile := filepath.Join(outputDir, cap.SourceFile+".yaml")

	if dryRun {
		fmt.Printf("=== %s ===\n%s\n", schemaFile, schema)
		return nil
	}

	if err := os.WriteFile(schemaFile, schema, 0600); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	if verbose {
		fmt.Printf("Generated XTP schema: %s\n", schemaFile)
	}
	return nil
}
