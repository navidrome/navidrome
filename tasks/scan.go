package tasks

import (
	"github.com/astaxie/beego/toolbox"
	"github.com/deluan/gosonic/scanner"
)

const TaskItunesScan = "iTunes Library Scanner"

func init() {
	scan := toolbox.NewTask(TaskItunesScan, "0/5 * * * * *", func() error {
		scanner.CheckForUpdates(false)
		return nil
	})

	toolbox.AddTask(TaskItunesScan, scan)
	toolbox.StartTask()
	defer toolbox.DeleteTask(TaskItunesScan)
}
