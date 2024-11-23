package cmd

import (
	"context"
	"encoding/gob"
	"os"

	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/utils/pl"
	"github.com/spf13/cobra"
)

var fullRescan bool

func init() {
	scanCmd.Flags().BoolVarP(&fullRescan, "full", "f", false, "check all subfolders, ignoring timestamps")
	rootCmd.AddCommand(scanCmd)
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan music folder",
	Long:  "Scan music folder for updates",
	Run: func(cmd *cobra.Command, args []string) {
		runScanner()
	},
}

func runScanner() {
	sqlDB := db.Db()
	ds := persistence.New(sqlDB)
	ctx := context.Background()
	progress := scanner.Scan(ctx, ds, artwork.NoopCacheWarmer(), fullRescan)
	encoder := gob.NewEncoder(os.Stdout)
	for status := range pl.ReadOrDone(ctx, progress) {
		err := encoder.Encode(status)
		if err != nil {
			log.Error(ctx, "Failed to encode status", err)
		}
	}

	if fullRescan {
		log.Info("Finished full rescan")
	} else {
		log.Info("Finished rescan")
	}
}
