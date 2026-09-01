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
		Long:  "Search index maintenance operations",
	}

	searchRebuildCmd = &cobra.Command{
		Use:   "rebuild",
		Short: "Rebuild the full-text search index",
		Long: "Drop and rebuild the full-text search index from the library data. Fixes a corrupted " +
			"or desynced search index without any data loss. This must be done offline",
		Run: func(cmd *cobra.Command, _ []string) {
			runSearchRebuild(cmd.Context())
		},
	}
)

func runSearchRebuild(ctx context.Context) {
	existingDBPath()

	if !searchRebuildForce && !confirmYES("This will rebuild the search index. Make sure Navidrome is not running.") {
		log.Warn("Rebuild cancelled")
		return
	}

	database := db.Db()
	defer db.Close(ctx)

	fmt.Println("Rebuilding the search index...")
	if err := db.RebuildFTS(ctx, database); err != nil {
		log.Fatal("Error rebuilding the search index", err)
	}
	if err := db.VerifyFTS(ctx, database); err != nil {
		log.Fatal("The search index still reports problems after the rebuild", err)
	}
	fmt.Println("Search index rebuilt successfully.")
}
