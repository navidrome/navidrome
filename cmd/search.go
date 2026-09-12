package cmd

import (
	"context"
	"fmt"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/spf13/cobra"
)

var searchRebuildForce bool

func init() {
	rootCmd.AddCommand(searchRoot)

	searchRebuildCmd.Flags().BoolVarP(&searchRebuildForce, "force", "f", false, "bypass rebuild confirmation")
	searchRoot.AddCommand(searchRebuildCmd)
}

var (
	searchRoot = &cobra.Command{
		Use:   "search",
		Short: "Search index maintenance",
	}

	searchRebuildCmd = &cobra.Command{
		Use:   "rebuild",
		Short: "Rebuild the full-text search index",
		Long: "Drop and rebuild the full-text search index from the library data. Fixes a corrupted " +
			"or desynced search index without any data loss. Note that 'navidrome doctor' detects a " +
			"corrupted index, but cannot tell when the index has merely drifted out of sync with the " +
			"library. This must be done offline",
		Run: func(cmd *cobra.Command, _ []string) {
			runSearchRebuild(cmd.Context())
		},
	}
)

func runSearchRebuild(ctx context.Context) {
	requireExistingDB()

	if !searchRebuildForce && !confirmYES("This will rebuild the search index. Make sure Navidrome is not running.") {
		log.Warn("Rebuild cancelled")
		return
	}

	fmt.Println("Rebuilding the search index...")
	err := db.RebuildFTS(ctx, db.Db())
	db.Close(ctx)
	if err != nil {
		log.Fatal("Error rebuilding the search index", err)
	}
	fmt.Println("Search index rebuilt successfully.")
}
