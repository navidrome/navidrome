package main

import (
	"fmt"
	"strings"

	"github.com/deluan/navidrome/static"
)

var (
	// This will be set in build time. If not, version will be set to "dev"
	gitTag string
	gitSha string
)

// Formats:
// dev
// v0.2.0 (5b84188)
// master (9ed35cb)
func getVersion() string {
	if gitSha == "" {
		return "dev"
	}
	return fmt.Sprintf("%s (%s)", gitTag, gitSha)
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
