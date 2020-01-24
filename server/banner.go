package server

import (
	"encoding/base64"
	"fmt"
)

// http://patorjk.com/software/taag/#p=display&f=Doom&t=Navidrome
// Need to be Base64 encoded, as it contains a lot of escaping chars.
// May try to find another way to do it without an external file
const encodedBanner = "IF8gICBfICAgICAgICAgICAgIF8gICAgIF8gICAgICAgICAgICAgICAgICAgICAgICAgIAp8IFwgfCB8ICAgICAgICAgIC" +
	"AoXykgICB8IHwgICAgICAgICAgICAgICAgICAgICAgICAgCnwgIFx8IHwgX18gX19fICAgX19fICBfX3wgfF8gX18gX19fICBfIF9fIF9fXyAgIF" +
	"9fXyAKfCAuIGAgfC8gX2AgXCBcIC8gLyB8LyBfYCB8ICdfXy8gXyBcfCAnXyBgIF8gXCAvIF8gXAp8IHxcICB8IChffCB8XCBWIC98IHwgKF98IH" +
	"wgfCB8IChfKSB8IHwgfCB8IHwgfCAgX18vClxffCBcXy9cX18sX3wgXF8vIHxffFxfXyxffF98ICBcX19fL3xffCB8X3wgfF98XF9fX3w="

const banner = `%s
                                       Version %s

`

func ShowBanner() {
	decodedBanner, _ := base64.StdEncoding.DecodeString(encodedBanner)
	fmt.Printf(banner, string(decodedBanner), Version)
}
