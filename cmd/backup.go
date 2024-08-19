package cmd

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/spf13/cobra"
)

var (
	backupPath string
)

func init() {
	backupCmd.Flags().StringVarP(&backupPath, "backup", "b", "", "directory to manually make backup")
	rootCmd.AddCommand(backupCmd)
}

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup database",
	Long:  "Manually backup Navidrome database. This will ignore BackupCount",
	Run: func(cmd *cobra.Command, _ []string) {
		runBackup()
	},
}

func runBackup() {
	conf.Server.Backup.Bypass = true
	if backupPath != "" {
		conf.Server.Backup.Path = backupPath
	}

	idx := strings.LastIndex(conf.Server.DbPath, "?")
	var path string

	if idx == -1 {
		path = conf.Server.DbPath
	} else {
		path = conf.Server.DbPath[:idx]
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Fatal("No existing database", "path", path)
		return
	}

	database := db.Db()
	start := time.Now()
	err := database.Backup(context.Background())
	if err != nil {
		log.Fatal("Error backing up database", "backup path", conf.Server.BasePath, err)
	}

	elapsed := time.Since(start)

	log.Info("Backup complete", "elapsed", elapsed)
}
