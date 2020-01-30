package main

import (
	"fmt"
	"strings"

	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/static"
)

func getBanner() string {
	data, _ := static.Asset("banner.txt")
	return strings.TrimSuffix(string(data), "\n")
}

func ShowBanner() {
	version := "Version: " + consts.Version()
	padding := strings.Repeat(" ", 52-len(version))
	fmt.Printf("%s%s%s\n\n", getBanner(), padding, version)
}
