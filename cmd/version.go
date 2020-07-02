package cmd

import (
	"fmt"

	"github.com/deluan/navidrome/consts"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print Navidrome's version",
	Long:  `All software has versions. This is Navidrome's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(consts.Version())
	},
}
