package consts

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/deluan/navidrome/static"
)

func getBanner() string {
	data, _ := static.Asset("banner.txt")
	return strings.TrimRightFunc(string(data), unicode.IsSpace)
}

func Banner() string {
	version := "Version: " + Version()
	padding := strings.Repeat(" ", 52-len(version))
	return fmt.Sprintf("%s\n%s%s\n", getBanner(), padding, version)
}
