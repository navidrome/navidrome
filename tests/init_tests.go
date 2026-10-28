package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/SunilSherkhane/navidrome/conf"
	_ "github.com/SunilSherkhane/navidrome/conf/mime" // registers mime_types.yaml, so tests see the same image types as the server
	"github.com/SunilSherkhane/navidrome/log"
)

var once sync.Once

func Init(t testing.TB, skipOnShort bool) {
	if skipOnShort && testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	once.Do(func() {
		_, file, _, _ := runtime.Caller(0)
		appPath, _ := filepath.Abs(filepath.Join(filepath.Dir(file), ".."))
		confPath, _ := filepath.Abs(filepath.Join(appPath, "tests", "navidrome-test.toml"))
		println("Loading test configuration file from " + confPath)
		_ = os.Chdir(appPath)
		conf.LoadFromFile(confPath)

		noLog := os.Getenv("NOLOG")
		if noLog != "" {
			log.SetLevel(log.LevelError)
		}
	})
}
