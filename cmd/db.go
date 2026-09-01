package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/spf13/cobra"
)

var (
	repairForce   bool
	repairRebuild bool
)

func init() {
	rootCmd.AddCommand(dbRoot)

	repairCmd.Flags().BoolVarP(&repairForce, "force", "f", false, "bypass repair confirmation")
	repairCmd.Flags().BoolVar(&repairRebuild, "rebuild", false, "rebuild the search index even if no corruption is found")
	dbRoot.AddCommand(repairCmd)
}

var (
	dbRoot = &cobra.Command{
		Use:   "db",
		Short: "Database maintenance",
		Long:  "Database maintenance operations",
	}

	repairCmd = &cobra.Command{
		Use:   "repair",
		Short: "Check database integrity and repair the search index",
		Long: "Check database integrity and rebuild the full-text search index if it is corrupted. " +
			"This must be done offline",
		Run: func(cmd *cobra.Command, _ []string) {
			runRepair(cmd.Context())
		},
	}
)

func runRepair(ctx context.Context) {
	existingDBPath()

	if !repairForce && !confirmYES("This will check the database and may rebuild the search index. Make sure Navidrome is not running.") {
		log.Warn("Repair cancelled")
		return
	}

	database := db.Db()
	defer db.Close(ctx)

	fmt.Println("Checking database integrity...")
	issues, err := db.IntegrityCheck(ctx, database)
	if err != nil {
		fmt.Println("The integrity check could not complete: " + err.Error())
		fmt.Println("Restore a backup (navidrome backup restore), or try SQLite's '.recover' command.")
		os.Exit(1)
	}

	if len(issues) == 0 {
		fmt.Println("Database integrity check passed.")
		if !repairRebuild {
			fmt.Println("Nothing to repair. Use --rebuild to rebuild the search index anyway.")
			return
		}
	} else {
		fmt.Printf("Integrity check reported %d issue(s):\n", len(issues))
		for _, issue := range issues {
			fmt.Println("  " + issue)
		}
		if !db.IsFTSCorruptionOnly(issues) {
			fmt.Println("Corruption is not limited to the search index, and cannot be repaired by this command.")
			fmt.Println("Restore a backup (navidrome backup restore), or try SQLite's '.recover' command.")
			os.Exit(1)
		}
		fmt.Println("Corruption is limited to the search index, which can be safely rebuilt.")
	}

	fmt.Println("Rebuilding the search index...")
	if err := db.RebuildFTS(ctx, database); err != nil {
		log.Fatal("Error rebuilding the search index", err)
	}
	if err := db.VerifyFTS(ctx, database); err != nil {
		log.Fatal("The search index still reports problems after the rebuild", err)
	}
	fmt.Println("Search index rebuilt successfully.")
}
