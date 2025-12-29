// hostgen generates Extism host function wrappers from annotated Go interfaces.
//
// Usage:
//
//	hostgen -input=./plugins/host -output=./plugins/host
//
// Flags:
//
//	-input       Input directory containing Go source files with annotated interfaces
//	-output      Output directory for generated files (default: same as input)
//	-package     Output package name (default: inferred from output directory)
//	-host-only   Generate only host-side code (default: false)
//	-plugin-only Generate only plugin/client-side code (default: false)
//	-go          Generate Go client wrappers (default: true when not using -python/-rust)
//	-python      Generate Python client wrappers (default: false)
//	-rust        Generate Rust client wrappers (default: false)
//	-v           Verbose output
//	-dry-run     Preview generated code without writing files
package main

import (
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/plugins/cmd/hostgen/internal"
)

func main() {
	var (
		inputDir   = flag.String("input", ".", "Input directory containing Go source files")
		outputDir  = flag.String("output", "", "Output directory for generated files (default: same as input)")
		pkgName    = flag.String("package", "", "Output package name (default: inferred from output directory)")
		hostOnly   = flag.Bool("host-only", false, "Generate only host-side code")
		pluginOnly = flag.Bool("plugin-only", false, "Generate only plugin/client-side code")
		goClient   = flag.Bool("go", false, "Generate Go client wrappers")
		pyClient   = flag.Bool("python", false, "Generate Python client wrappers")
		rsClient   = flag.Bool("rust", false, "Generate Rust client wrappers")
		verbose    = flag.Bool("v", false, "Verbose output")
		dryRun     = flag.Bool("dry-run", false, "Preview generated code without writing files")
	)
	flag.Parse()

	// Validate conflicting flags
	if *hostOnly && *pluginOnly {
		fmt.Fprintf(os.Stderr, "Error: -host-only and -plugin-only cannot be used together\n")
		os.Exit(1)
	}

	if *outputDir == "" {
		*outputDir = *inputDir
	}

	// Resolve absolute paths
	absInput, err := filepath.Abs(*inputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving input path: %v\n", err)
		os.Exit(1)
	}
	absOutput, err := filepath.Abs(*outputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving output path: %v\n", err)
		os.Exit(1)
	}

	// Infer package name if not provided
	if *pkgName == "" {
		*pkgName = filepath.Base(absOutput)
	}

	// Determine what to generate
	generateHost := !*pluginOnly
	// Default: generate Go clients if no language flag is specified
	// If -python or -rust is specified without -go, only generate those
	// If -go is specified, generate Go
	// If multiple are specified, generate all specified
	anyLangFlag := *goClient || *pyClient || *rsClient
	generateGoClient := !*hostOnly && (*goClient || !anyLangFlag)
	generatePyClient := !*hostOnly && *pyClient
	generateRsClient := !*hostOnly && *rsClient

	if *verbose {
		fmt.Printf("Input directory: %s\n", absInput)
		fmt.Printf("Output directory: %s\n", absOutput)
		fmt.Printf("Package name: %s\n", *pkgName)
		fmt.Printf("Generate host code: %v\n", generateHost)
		fmt.Printf("Generate Go client code: %v\n", generateGoClient)
		fmt.Printf("Generate Python client code: %v\n", generatePyClient)
		fmt.Printf("Generate Rust client code: %v\n", generateRsClient)
	}

	// Parse source files
	services, err := internal.ParseDirectory(absInput)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing source files: %v\n", err)
		os.Exit(1)
	}

	if len(services) == 0 {
		if *verbose {
			fmt.Println("No host services found")
		}
		return
	}

	if *verbose {
		fmt.Printf("Found %d host service(s)\n", len(services))
		for _, svc := range services {
			fmt.Printf("  - %s (%d methods)\n", svc.Name, len(svc.Methods))
		}
	}

	// Generate code for each service
	for _, svc := range services {
		// Generate host-side code
		if generateHost {
			if err := generateHostCode(svc, *pkgName, absOutput, *dryRun, *verbose); err != nil {
				fmt.Fprintf(os.Stderr, "Error generating host code for %s: %v\n", svc.Name, err)
				os.Exit(1)
			}
		}

		// Generate Go client-side code
		if generateGoClient {
			if err := generateGoClientCode(svc, absOutput, *dryRun, *verbose); err != nil {
				fmt.Fprintf(os.Stderr, "Error generating Go client code for %s: %v\n", svc.Name, err)
				os.Exit(1)
			}
		}

		// Generate Python client-side code
		if generatePyClient {
			if err := generatePythonClientCode(svc, absOutput, *dryRun, *verbose); err != nil {
				fmt.Fprintf(os.Stderr, "Error generating Python client code for %s: %v\n", svc.Name, err)
				os.Exit(1)
			}
		}

		// Generate Rust client-side code
		if generateRsClient {
			if err := generateRustClientCode(svc, absOutput, *dryRun, *verbose); err != nil {
				fmt.Fprintf(os.Stderr, "Error generating Rust client code for %s: %v\n", svc.Name, err)
				os.Exit(1)
			}
		}
	}

	// Generate Rust lib.rs to expose all modules
	if generateRsClient && len(services) > 0 {
		if err := generateRustLibFile(services, absOutput, *dryRun, *verbose); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating Rust lib.rs: %v\n", err)
			os.Exit(1)
		}
	}
}

// generateHostCode generates host-side code for a service.
func generateHostCode(svc internal.Service, pkgName, outputDir string, dryRun, verbose bool) error {
	code, err := internal.GenerateHost(svc, pkgName)
	if err != nil {
		return fmt.Errorf("generating code: %w", err)
	}

	formatted, err := format.Source(code)
	if err != nil {
		return fmt.Errorf("formatting code: %w\nRaw code:\n%s", err, code)
	}

	outputFile := filepath.Join(outputDir, svc.OutputFileName())

	if dryRun {
		fmt.Printf("=== %s ===\n%s\n", outputFile, formatted)
		return nil
	}

	if err := os.WriteFile(outputFile, formatted, 0600); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	if verbose {
		fmt.Printf("Generated host code: %s\n", outputFile)
	}
	return nil
}

// generateGoClientCode generates Go client-side code for a service.
func generateGoClientCode(svc internal.Service, outputDir string, dryRun, verbose bool) error {
	code, err := internal.GenerateClientGo(svc)
	if err != nil {
		return fmt.Errorf("generating code: %w", err)
	}

	formatted, err := format.Source(code)
	if err != nil {
		return fmt.Errorf("formatting code: %w\nRaw code:\n%s", err, code)
	}

	// Client code goes in go/ subdirectory
	clientDir := filepath.Join(outputDir, "go")
	clientFile := filepath.Join(clientDir, "nd_host_"+strings.ToLower(svc.Name)+".go")

	if dryRun {
		fmt.Printf("=== %s ===\n%s\n", clientFile, formatted)
		return nil
	}

	// Create go/ subdirectory if needed
	if err := os.MkdirAll(clientDir, 0755); err != nil {
		return fmt.Errorf("creating client directory: %w", err)
	}

	if err := os.WriteFile(clientFile, formatted, 0600); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	if verbose {
		fmt.Printf("Generated Go client code: %s\n", clientFile)
	}
	return nil
}

// generatePythonClientCode generates Python client-side code for a service.
func generatePythonClientCode(svc internal.Service, outputDir string, dryRun, verbose bool) error {
	code, err := internal.GenerateClientPython(svc)
	if err != nil {
		return fmt.Errorf("generating code: %w", err)
	}

	// Python code goes in python/ subdirectory
	clientDir := filepath.Join(outputDir, "python")
	clientFile := filepath.Join(clientDir, "nd_host_"+strings.ToLower(svc.Name)+".py")

	if dryRun {
		fmt.Printf("=== %s ===\n%s\n", clientFile, code)
		return nil
	}

	// Create python/ subdirectory if needed
	if err := os.MkdirAll(clientDir, 0755); err != nil {
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

	// Rust code goes in rust/ subdirectory
	clientDir := filepath.Join(outputDir, "rust")
	clientFile := filepath.Join(clientDir, "nd_host_"+strings.ToLower(svc.Name)+".rs")

	if dryRun {
		fmt.Printf("=== %s ===\n%s\n", clientFile, code)
		return nil
	}

	// Create rust/ subdirectory if needed
	if err := os.MkdirAll(clientDir, 0755); err != nil {
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

	clientDir := filepath.Join(outputDir, "rust")
	libFile := filepath.Join(clientDir, "lib.rs")

	if dryRun {
		fmt.Printf("=== %s ===\n%s\n", libFile, code)
		return nil
	}

	// Create rust/ subdirectory if needed
	if err := os.MkdirAll(clientDir, 0755); err != nil {
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
