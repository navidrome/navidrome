package cmd

import (
	"context"
	"path/filepath"
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

	backupCmd.Flags().StringVarP(&backupDir, "backup-dir", "d", "", "directory to manually make backup")
	backupRoot.AddCommand(backupCmd)

	pruneCmd.Flags().StringVarP(&backupDir, "backup-dir", "d", "", "directory holding Navidrome backups")
	pruneCmd.Flags().IntVarP(&backupCount, "keep-count", "k", -1, "specify the number of backups to keep. 0 remove ALL backups, and negative values mean to use the default from configuration")
	pruneCmd.Flags().BoolVarP(&force, "force", "f", false, "bypass warning when backup count is zero")
	backupRoot.AddCommand(pruneCmd)

	restoreCommand.Flags().StringVarP(&restorePath, "backup-file", "b", "", "file name of the backup database to restore (resolved against the backup directory unless it is an absolute path)")
	restoreCommand.Flags().BoolVarP(&force, "force", "f", false, "bypass restore warning")
	_ = restoreCommand.MarkFlagRequired("backup-file")
	backupRoot.AddCommand(restoreCommand)
}

var (
	backupRoot = &cobra.Command{
		Use:     "backup",
		Aliases: []string{"bkp"},
		Short:   "Create, restore and prune database backups",
		Long:    "Create, restore and prune database backups",
	}

	backupCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a backup database",
		Long:  "Manually backup Navidrome database. This will ignore BackupCount",
		Run: func(cmd *cobra.Command, _ []string) {
			runBackup(cmd.Context())
		},
	}

	pruneCmd = &cobra.Command{
		Use:   "prune",
		Short: "Prune database backups",
		Long:  "Manually prune database backups according to backup rules",
		Run: func(cmd *cobra.Command, _ []string) {
			runPrune(cmd.Context())
		},
	}

	restoreCommand = &cobra.Command{
		Use:   "restore",
		Short: "Restore Navidrome database",
		Long:  "Restore Navidrome database from a backup. This must be done offline",
		Run: func(cmd *cobra.Command, _ []string) {
			runRestore(cmd.Context())
		},
	}
)

func runBackup(ctx context.Context) {
	if backupDir != "" {
		conf.Server.Backup.Path = conf.NewDir(backupDir)
	}

	requireExistingDB()

	start := time.Now()
	path, err := db.Backup(ctx)
	if err != nil {
		log.Fatal("Error backing up database", "backupPath", conf.Server.Backup.Path, err)
	}

	elapsed := time.Since(start)
	log.Info("Backup complete", "elapsed", elapsed, "path", path)
}

func runPrune(ctx context.Context) {
	if backupDir != "" {
		conf.Server.Backup.Path = conf.NewDir(backupDir)
	}

	if backupCount != -1 {
		conf.Server.Backup.Count = backupCount
	}

	if conf.Server.Backup.Count == 0 && !force && !confirmYES("Warning: pruning ALL backups") {
		log.Warn("Prune cancelled")
		return
	}

	requireExistingDB()

	start := time.Now()
	count, err := db.Prune(ctx)
	if err != nil {
		log.Fatal("Error pruning database", "backupPath", conf.Server.Backup.Path, err)
	}

	elapsed := time.Since(start)

	log.Info("Prune complete", "elapsed", elapsed, "successfully pruned", count)
}

func runRestore(ctx context.Context) {
	requireExistingDB()

	// A relative --backup-file is resolved against Backup.Path, the same folder
	// `backup create` writes to. Without this, the value was treated as relative
	// to the working directory, where the file does not exist.
	if !filepath.IsAbs(restorePath) {
		backupPath, err := conf.Server.Backup.Path.Path()
		if err != nil {
			log.Fatal("Backup directory not available", "backupPath", conf.Server.Backup.Path, err)
			return
		}
		restorePath = filepath.Join(backupPath, restorePath)
	}

	if !force && !confirmYES("Warning: restoring the Navidrome database should only be done offline, especially if your backup is very old.") {
		log.Warn("Restore cancelled")
		return
	}

	start := time.Now()
	err := db.Restore(ctx, restorePath)
	if err != nil {
		log.Fatal("Error restoring database", "backupFile", restorePath, err)
	}

	elapsed := time.Since(start)
	log.Info("Restore complete", "elapsed", elapsed)
}
