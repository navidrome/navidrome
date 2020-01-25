package main

import (
	"fmt"
	"strings"

	"github.com/deluan/navidrome/static"
)

var (
	// This will be set in build time. If not, version will be set to "dev"
	gitBranch string
	gitTag    string
	gitHash   string
	gitCount  string
)

// Formats:
// dev
// v0.2.0 (596-5b84188)
// master (600-9ed35cb)
func getVersion() string {
	if gitHash == "" {
		return "dev"
	}
	version := fmt.Sprintf(" (%s-%s)", gitCount, gitHash)
	if gitTag != "" {
		return gitTag + version
	}
	return gitBranch + version
}

func getBanner() string {
	data, _ := static.Asset("banner.txt")
	return strings.TrimSuffix(string(data), "\n")
}

func ShowBanner() {
	version := "Version: " + getVersion()
	padding := strings.Repeat(" ", 52-len(version))
	fmt.Printf("%s%s%s\n\n", getBanner(), padding, version)
}
