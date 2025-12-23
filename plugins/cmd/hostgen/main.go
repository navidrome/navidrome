// hostgen generates Extism host function wrappers from annotated Go interfaces.
//
// Usage:
//
//	hostgen -input=./plugins/host -output=./plugins/host
//
// Flags:
//
//	-input    Input directory containing Go source files with annotated interfaces
//	-output   Output directory for generated files (default: same as input)
//	-package  Output package name (default: inferred from output directory)
//	-v        Verbose output
//	-dry-run  Preview generated code without writing files
package main

import (
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/plugins/cmd/hostgen/internal"
)

func main() {
	var (
		inputDir  = flag.String("input", ".", "Input directory containing Go source files")
		outputDir = flag.String("output", "", "Output directory for generated files (default: same as input)")
		pkgName   = flag.String("package", "", "Output package name (default: inferred from output directory)")
		verbose   = flag.Bool("v", false, "Verbose output")
		dryRun    = flag.Bool("dry-run", false, "Preview generated code without writing files")
	)
	flag.Parse()

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

	if *verbose {
		fmt.Printf("Input directory: %s\n", absInput)
		fmt.Printf("Output directory: %s\n", absOutput)
		fmt.Printf("Package name: %s\n", *pkgName)
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
		code, err := internal.GenerateService(svc, *pkgName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating code for %s: %v\n", svc.Name, err)
			os.Exit(1)
		}

		// Format the generated code
		formatted, err := format.Source(code)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting generated code for %s: %v\n", svc.Name, err)
			fmt.Fprintf(os.Stderr, "Raw code:\n%s\n", code)
			os.Exit(1)
		}

		outputFile := filepath.Join(absOutput, svc.OutputFileName())

		if *dryRun {
			fmt.Printf("=== %s ===\n%s\n", outputFile, formatted)
			continue
		}

		if err := os.WriteFile(outputFile, formatted, 0600); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", outputFile, err)
			os.Exit(1)
		}

		if *verbose {
			fmt.Printf("Generated %s\n", outputFile)
		}
	}
}
