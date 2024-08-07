package cmd

import (
	"context"

	"github.com/navidrome/navidrome/log"
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
	scanner := GetScanner(context.Background())
	_ = scanner.RescanAll(context.Background(), fullRescan)
	if fullRescan {
		log.Info("Finished full rescan")
	} else {
		log.Info("Finished rescan")
	}
}
