package cmd

import (
	"time"

	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/scanner"
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

func waitScanToFinish(scanner scanner.Scanner) {
	time.Sleep(500 * time.Millisecond)
	ticker := time.Tick(100 * time.Millisecond)
	for {
		if !scanner.Scanning() {
			return
		}
		<-ticker
	}
}

func runScanner() {
	scanner := GetScanner()
	go func() { _ = scanner.Start(0) }()
	scanner.RescanAll(fullRescan)
	waitScanToFinish(scanner)
	scanner.Stop()
	if fullRescan {
		log.Info("Finished full rescan")
	} else {
		log.Info("Finished rescan")
	}
}
