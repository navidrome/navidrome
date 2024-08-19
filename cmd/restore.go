package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/spf13/cobra"
)

var (
	ignoreRestoreWarning bool
	restoredDbPath       string
)

func init() {
	restoreCommand.Flags().StringVarP(&restoredDbPath, "backup", "b", "", "path to backup database restore")
	restoreCommand.Flags().BoolVarP(&ignoreRestoreWarning, "force", "f", false, "bypass restore warning")
	_ = restoreCommand.MarkFlagRequired("backup")
	rootCmd.AddCommand(restoreCommand)
}

var restoreCommand = &cobra.Command{
	Use:   "restore",
	Short: "Restore Navidrome database",
	Long:  "Restore Navidrome database from a backup. This must be done offline",
	Run: func(cmd *cobra.Command, _ []string) {
		runRestore()
	},
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

	if !ignoreRestoreWarning {
		fmt.Println("Warning: restoring the Navidrome database should only be done offline, especially if your backup is very old.")
		fmt.Printf("Please enter any character: ")
		var input string
		_, err := fmt.Scanln(&input)

		if input == "" || err != nil {
			log.Info("Restore cancelled")
			return
		}
	}

	database := db.Db()
	err := database.Restore(context.Background(), restoredDbPath)
	if err != nil {
		log.Fatal("Error backing up database", "backup path", conf.Server.BasePath, err)
	}

	log.Info("Restore complete")
}
