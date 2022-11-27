package resources

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/navidrome/navidrome/consts"
)

func loadBanner() string {
	data, _ := Asset("banner.txt")
	return strings.TrimRightFunc(string(data), unicode.IsSpace)
}

func Banner() string {
	version := "Version: " + consts.Version
	return fmt.Sprintf("%s\n%52s\n", loadBanner(), version)
}
