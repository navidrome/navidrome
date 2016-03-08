package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/utils"
)

func Init(t *testing.T, skipOnShort bool) {
	if skipOnShort && testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	_, file, _, _ := runtime.Caller(0)
	appPath, _ := filepath.Abs(filepath.Join(filepath.Dir(file), ".."))
	beego.TestBeegoInit(appPath)

	noLog := os.Getenv("NOLOG")
	if noLog != "" {
		beego.SetLevel(beego.LevelError)
	}
	utils.Graph.Finalize()
}
