package conf

import (
	"fmt"
	"os"

	"github.com/koding/multiconfig"
)

type sonic struct {
	Port        string `default:"4533"`
	MusicFolder string `default:"./iTunes1.xml"`
	DbPath      string `default:"./devDb"`

	IgnoredArticles string `default:"The El La Los Las Le Les Os As O A"`
	IndexGroups     string `default:"A B C D E F G H I J K L M N O P Q R S T U V W X-Z(XYZ) [Unknown]([)"`

	DisableAuthentication bool   `default:"false"`
	User                  string `default:"anyone"`
	Password              string `default:"wordpass"`

	DisableDownsampling bool   `default:"false"`
	DownsampleCommand   string `default:"ffmpeg -i %s -map 0:0 -b:a %bk -v 0 -f mp3 -"`
	PlsIgnoreFolders    bool   `default:"true"`
	PlsIgnoredPatterns  string `default:"^iCloud;\\~"`
}

var Sonic *sonic

func LoadFromFlags() {
	l := &multiconfig.FlagLoader{}
	l.Load(Sonic)
}

func LoadFromEnv() {
	port := os.Getenv("PORT")
	if port != "" {
		Sonic.Port = port
	}
	l := &multiconfig.EnvironmentLoader{}
	l.Load(Sonic)
}

func LoadFromTags() {
	l := &multiconfig.TagLoader{}
	l.Load(Sonic)
}

func LoadFromFile(tomlFile string) {
	l := &multiconfig.TOMLLoader{Path: tomlFile}
	err := l.Load(Sonic)
	if err != nil {
		fmt.Printf("Error loading %s: %v\n", tomlFile, err)
	}
}

func LoadFromLocalFile() {
	if _, err := os.Stat("./sonic.toml"); err == nil {
		LoadFromFile("./sonic.toml")
	}
}

func Load() {
	LoadFromLocalFile()
	LoadFromEnv()
	LoadFromFlags()
}

func init() {
	Sonic = new(sonic)
	LoadFromTags()
}
