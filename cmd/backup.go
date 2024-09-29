package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/spf13/cobra"
)

var (
	backupCount int
	backupDir   string
	force       bool
	restorePath string
)

func init() {
	rootCmd.AddCommand(backupRoot)

	backupCmd.Flags().StringVarP(&backupDir, "backup-dir", "b", "", "directory to manually make backup")
	backupRoot.AddCommand(backupCmd)

	pruneCmd.Flags().StringVarP(&backupDir, "backup-directory", "b", "", "directory holding Navidrome backups")
	pruneCmd.Flags().IntVarP(&backupCount, "keep-count", "k", -1, "specify the number of backups to keep. 0 preserve no backups, and negative values mean to use the default from configuration")
	pruneCmd.Flags().BoolVarP(&force, "force", "f", false, "bypass warning when backup count is zero")
	backupRoot.AddCommand(pruneCmd)

	restoreCommand.Flags().StringVarP(&restorePath, "backup-path", "b", "", "path of backup database restore")
	restoreCommand.Flags().BoolVarP(&force, "force", "f", false, "bypass restore warning")
	_ = restoreCommand.MarkFlagRequired("backup-path")
	backupRoot.AddCommand(restoreCommand)
}

var (
	backupRoot = &cobra.Command{
		Use:   "backup",
		Short: "Backup/restore/prune database",
		Long:  "Backup/restore/prune database",
	}

	backupCmd = &cobra.Command{
		Use:   "backup",
		Short: "Backup database",
		Long:  "Manually backup Navidrome database. This will ignore BackupCount",
		Run: func(cmd *cobra.Command, _ []string) {
			runBackup()
		},
	}

	pruneCmd = &cobra.Command{
		Use:   "prune",
		Short: "Prune database backups",
		Long:  "Manually prune database backups according to backup rules",
		Run: func(cmd *cobra.Command, _ []string) {
			runPrune()
		},
	}

	restoreCommand = &cobra.Command{
		Use:   "restore",
		Short: "Restore Navidrome database",
		Long:  "Restore Navidrome database from a backup. This must be done offline",
		Run: func(cmd *cobra.Command, _ []string) {
			runRestore()
		},
	}
)

func runBackup() {
	conf.Server.Backup.Bypass = true
	if backupDir != "" {
		conf.Server.Backup.Path = backupDir
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

func runPrune() {
	if backupDir != "" {
		conf.Server.Backup.Path = backupDir
	}

	if backupCount != -1 {
		conf.Server.Backup.Count = backupCount
	}

	if conf.Server.Backup.Count == 0 && !force {
		fmt.Println("Warning: pruning ALL backups")
		fmt.Printf("Please enter YES (all caps) to continue: ")
		var input string
		_, err := fmt.Scanln(&input)

		if input != "YES" || err != nil {
			log.Warn("Restore cancelled")
			return
		}
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
	count, err := database.Prune(context.Background())
	if err != nil {
		log.Fatal("Error pruning up database", "backup path", conf.Server.BasePath, err)
	}

	elapsed := time.Since(start)

	log.Info("Prune complete", "elapsed", elapsed, "successfully pruned", count)
}

func runRestore() {
	conf.Server.Backup.Bypass = true

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

	if !force {
		fmt.Println("Warning: restoring the Navidrome database should only be done offline, especially if your backup is very old.")
		fmt.Printf("Please enter YES (all caps) to continue: ")
		var input string
		_, err := fmt.Scanln(&input)

		if input != "YES" || err != nil {
			log.Warn("Restore cancelled")
			return
		}
	}

	database := db.Db()
	start := time.Now()
	err := database.Restore(context.Background(), restorePath)
	if err != nil {
		log.Fatal("Error backing up database", "backup path", conf.Server.BasePath, err)
	}

	elapsed := time.Since(start)
	log.Info("Restore complete", "elapsed", elapsed)
}
