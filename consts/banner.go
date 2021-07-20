package consts

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/navidrome/navidrome/resources"
)

func loadBanner() string {
	data, _ := resources.Asset("banner.txt")
	return strings.TrimRightFunc(string(data), unicode.IsSpace)
}

func Banner() string {
	version := "Version: " + Version()
	return fmt.Sprintf("%s\n%52s\n", loadBanner(), version)
}
