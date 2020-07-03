package cmd

import (
	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/log"
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
	scanner := CreateScanner(conf.Server.MusicFolder)
	err := scanner.RescanAll(fullRescan)
	if err != nil {
		log.Error("Error scanning media folder", "folder", conf.Server.MusicFolder, err)
	}
	if fullRescan {
		log.Info("Finished full rescan")
	} else {
		log.Info("Finished rescan")
	}
}
