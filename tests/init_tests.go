package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/astaxie/beego"
	"github.com/cloudsonic/sonic-server/conf"
	"github.com/cloudsonic/sonic-server/utils"
)

var initSync sync.Once

func Init(t *testing.T, skipOnShort bool) {
	if skipOnShort && testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	_, file, _, _ := runtime.Caller(0)
	appPath, _ := filepath.Abs(filepath.Join(filepath.Dir(file), ".."))
	confPath, _ := filepath.Abs(filepath.Join(appPath, "tests", "sonic-test.toml"))

	conf.LoadFromFile(confPath)

	initSync.Do(func() {
		beego.TestBeegoInit(appPath)
	})

	noLog := os.Getenv("NOLOG")
	if noLog != "" {
		beego.SetLevel(beego.LevelError)
	}
	utils.Graph.Finalize()
}
