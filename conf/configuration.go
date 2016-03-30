package conf

import (
	"fmt"
	"os"

	"github.com/koding/multiconfig"
)

type goSonic struct {
	Port        int    `default:"8080"`
	MusicFolder string `default:"./iTunes1.xml"`
	DbPath      string `default:"./devDb"`

	IgnoredArticles string `default:"The El La Los Las Le Les Os As O A"`
	IndexGroups     string `default:"A B C D E F G H I J K L M N O P Q R S T U V W X-Z(XYZ) [Unknown]([)"`

	User     string `default:"deluan"`
	Password string `default:"wordpass"`

	DisableDownsampling bool   `default:"false"`
	DisableValidation   bool   `default:"false"`
	DownsampleCommand   string `default:"ffmpeg -i %s -map 0:0 -b:a %bk -v 0 -f mp3 -"`
	PlsIgnoreFolders    bool   `default:"true"`
	PlsIgnoredPatterns  string `default:"^iCloud;^CDs para;^Skipped;Christian"`
	RunMode             string `default:"dev"`
}

var GoSonic *goSonic

func LoadFromFlags() {
	l := &multiconfig.FlagLoader{}
	l.Load(GoSonic)
}

func LoadFromFile(tomlFile string) {
	l := &multiconfig.TOMLLoader{Path: tomlFile}
	err := l.Load(GoSonic)
	if err != nil {
		fmt.Printf("Error loading %s: %v\n", tomlFile, err)
	}
}

func LoadFromLocalFile() {
	if _, err := os.Stat("./gosonic.toml"); err == nil {
		LoadFromFile("./gosonic.toml")
	}
}

func init() {
	GoSonic = new(goSonic)
	var l multiconfig.Loader
	l = &multiconfig.TagLoader{}
	l.Load(GoSonic)
	l = &multiconfig.EnvironmentLoader{}
	l.Load(GoSonic)
}
