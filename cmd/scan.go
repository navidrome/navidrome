package cmd

import (
	"context"
	"encoding/gob"
	"os"
	"strings"

	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/utils/pl"
	"github.com/spf13/cobra"
)

var (
	fullScan   bool
	subprocess bool
	targets    string
)

func init() {
	scanCmd.Flags().BoolVarP(&fullScan, "full", "f", false, "check all subfolders, ignoring timestamps")
	scanCmd.Flags().BoolVarP(&subprocess, "subprocess", "", false, "run as subprocess (internal use)")
	scanCmd.Flags().StringVarP(&targets, "targets", "t", "", "comma-separated list of libraryID:folderPath pairs (e.g., \"1:Music/Rock,1:Music/Jazz,2:Classical\")")
	rootCmd.AddCommand(scanCmd)
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan music folder",
	Long:  "Scan music folder for updates",
	Run: func(cmd *cobra.Command, args []string) {
		runScanner(cmd.Context())
	},
}

func trackScanInteractively(ctx context.Context, progress <-chan *scanner.ProgressInfo) {
	for status := range pl.ReadOrDone(ctx, progress) {
		if status.Warning != "" {
			log.Warn(ctx, "Scan warning", "error", status.Warning)
		}
		if status.Error != "" {
			log.Error(ctx, "Scan error", "error", status.Error)
		}
		// Discard the progress status, we only care about errors
	}

	if fullScan {
		log.Info("Finished full rescan")
	} else {
		log.Info("Finished rescan")
	}
}

func trackScanAsSubprocess(ctx context.Context, progress <-chan *scanner.ProgressInfo) {
	encoder := gob.NewEncoder(os.Stdout)
	for status := range pl.ReadOrDone(ctx, progress) {
		err := encoder.Encode(status)
		if err != nil {
			log.Error(ctx, "Failed to encode status", err)
		}
	}
}

func runScanner(ctx context.Context) {
	sqlDB := db.Db()
	defer db.Db().Close()
	ds := persistence.New(sqlDB)
	pls := core.NewPlaylists(ds)

	// Parse targets if provided
	var scanTargets []model.ScanTarget
	if targets != "" {
		var err error
		scanTargets, err = model.ParseTargets(strings.Split(targets, ","))
		if err != nil {
			log.Fatal(ctx, "Failed to parse targets", err)
		}
		log.Info(ctx, "Scanning specific folders", "numTargets", len(scanTargets))
	}

	progress, err := scanner.CallScan(ctx, ds, pls, fullScan, scanTargets)
	if err != nil {
		log.Fatal(ctx, "Failed to scan", err)
	}

	// Wait for the scanner to finish
	if subprocess {
		trackScanAsSubprocess(ctx, progress)
	} else {
		trackScanInteractively(ctx, progress)
	}
}
