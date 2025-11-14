package scanner

import (
	"bufio"
	"context"
	"io/fs"
	"path"
	"strings"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	ignore "github.com/sabhiram/go-gitignore"
)

// IgnoreChecker manages .ndignore patterns using a stack-based approach.
// Use Push() to add patterns when entering a folder, Pop() when leaving,
// and ShouldIgnore() to check if a path should be ignored.
type IgnoreChecker struct {
	fsys            fs.FS
	patternStack    [][]string        // Stack of patterns for each folder level
	currentPatterns []string          // Flattened current patterns
	matcher         *ignore.GitIgnore // Compiled matcher for current patterns
}

// newIgnoreChecker creates a new IgnoreChecker for the given filesystem.
func newIgnoreChecker(fsys fs.FS) *IgnoreChecker {
	return &IgnoreChecker{
		fsys:         fsys,
		patternStack: make([][]string, 0),
	}
}

// Push loads .ndignore patterns from the specified folder and adds them to the pattern stack.
// Use this when entering a folder during directory tree traversal.
func (ic *IgnoreChecker) Push(ctx context.Context, folder string) error {
	patterns := ic.loadPatternsFromFolder(ctx, folder)
	ic.patternStack = append(ic.patternStack, patterns)
	ic.rebuildCurrentPatterns()
	return nil
}

// Pop removes the most recent patterns from the stack.
// Use this when leaving a folder during directory tree traversal.
func (ic *IgnoreChecker) Pop() {
	if len(ic.patternStack) > 0 {
		ic.patternStack = ic.patternStack[:len(ic.patternStack)-1]
		ic.rebuildCurrentPatterns()
	}
}

// PushAllParents pushes patterns from root down to the target path.
// This is a convenience method for when you need to check a specific path
// without recursively walking the tree. It handles the common pattern of
// pushing all parent directories from root to the target.
// This method is optimized to compile patterns only once at the end.
func (ic *IgnoreChecker) PushAllParents(ctx context.Context, targetPath string) error {
	if targetPath == "." || targetPath == "" {
		// Simple case: just push root
		return ic.Push(ctx, ".")
	}

	// Load patterns for root
	patterns := ic.loadPatternsFromFolder(ctx, ".")
	ic.patternStack = append(ic.patternStack, patterns)

	// Load patterns for each parent directory
	currentPath := "."
	parts := strings.Split(path.Clean(targetPath), "/")
	for _, part := range parts {
		if part == "." || part == "" {
			continue
		}
		currentPath = path.Join(currentPath, part)
		patterns = ic.loadPatternsFromFolder(ctx, currentPath)
		ic.patternStack = append(ic.patternStack, patterns)
	}

	// Rebuild and compile patterns only once at the end
	ic.rebuildCurrentPatterns()
	return nil
}

// ShouldIgnore checks if the given path should be ignored based on the current patterns.
// Returns true if the path matches any ignore pattern, false otherwise.
func (ic *IgnoreChecker) ShouldIgnore(ctx context.Context, relPath string) bool {
	// Handle root/empty path - never ignore
	if relPath == "" || relPath == "." {
		return false
	}

	// If no patterns loaded, nothing to ignore
	if ic.matcher == nil {
		return false
	}

	matches := ic.matcher.MatchesPath(relPath)
	if matches {
		log.Trace(ctx, "Scanner: Ignoring entry matching .ndignore", "path", relPath)
	}
	return matches
}

// loadPatternsFromFolder reads the .ndignore file in the specified folder and returns the patterns.
// If the file doesn't exist, returns an empty slice.
// If the file exists but is empty, returns a pattern to ignore everything ("**/*").
func (ic *IgnoreChecker) loadPatternsFromFolder(ctx context.Context, folder string) []string {
	ignoreFilePath := path.Join(folder, consts.ScanIgnoreFile)
	var patterns []string

	// Check if .ndignore file exists
	if _, err := fs.Stat(ic.fsys, ignoreFilePath); err != nil {
		// No .ndignore file in this folder
		return patterns
	}

	// Read and parse the .ndignore file
	ignoreFile, err := ic.fsys.Open(ignoreFilePath)
	if err != nil {
		log.Warn(ctx, "Scanner: Error opening .ndignore file", "path", ignoreFilePath, err)
		return patterns
	}
	defer ignoreFile.Close()

	lineScanner := bufio.NewScanner(ignoreFile)
	for lineScanner.Scan() {
		line := strings.TrimSpace(lineScanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines, whitespace-only lines, and comments
		}
		patterns = append(patterns, line)
	}

	if err := lineScanner.Err(); err != nil {
		log.Warn(ctx, "Scanner: Error reading .ndignore file", "path", ignoreFilePath, err)
		return patterns
	}

	// If the .ndignore file is empty, ignore everything
	if len(patterns) == 0 {
		log.Trace(ctx, "Scanner: .ndignore file is empty, ignoring everything", "path", folder)
		patterns = []string{"**/*"}
	}

	return patterns
}

// rebuildCurrentPatterns flattens the pattern stack into currentPatterns and recompiles the matcher.
func (ic *IgnoreChecker) rebuildCurrentPatterns() {
	ic.currentPatterns = make([]string, 0)
	for _, patterns := range ic.patternStack {
		ic.currentPatterns = append(ic.currentPatterns, patterns...)
	}
	ic.compilePatterns()
}

// compilePatterns compiles the current patterns into a GitIgnore matcher.
func (ic *IgnoreChecker) compilePatterns() {
	if len(ic.currentPatterns) == 0 {
		ic.matcher = nil
		return
	}
	ic.matcher = ignore.CompileIgnoreLines(ic.currentPatterns...)
}
