package cmd

import (
	"context"

	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/scanner"
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
	s := scanner.GetLocalInstance(ctx, ds, artwork.NoopCacheWarmer())

	_ = s.ScanAll(ctx, fullRescan)
	if fullRescan {
		log.Info("Finished full rescan")
	} else {
		log.Info("Finished rescan")
	}
}
