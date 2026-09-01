package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/navidrome/navidrome/db"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(dbRoot)
	dbRoot.AddCommand(doctorCmd)
}

var (
	dbRoot = &cobra.Command{
		Use:   "db",
		Short: "Database maintenance",
		Long:  "Database maintenance operations",
	}

	doctorCmd = &cobra.Command{
		Use:   "doctor",
		Short: "Check the database for corruption and inconsistencies",
		Long: "Run read-only database health checks (integrity and foreign key checks) and " +
			"report what was found. This command never modifies the database",
		Run: func(cmd *cobra.Command, _ []string) {
			runDoctor(cmd.Context())
		},
	}
)

func runDoctor(ctx context.Context) {
	existingDBPath()

	database := db.Db()
	defer db.Close(ctx)

	healthy := true

	fmt.Println("Checking database integrity...")
	issues, err := db.IntegrityCheck(ctx, database)
	switch {
	case err != nil:
		fmt.Println("The integrity check could not complete: " + err.Error())
		fmt.Println("Restore a backup (navidrome backup restore), or try SQLite's '.recover' command.")
		os.Exit(1)
	case len(issues) == 0:
		fmt.Println("Integrity check passed.")
	default:
		healthy = false
		fmt.Printf("Integrity check reported %d issue(s):\n", len(issues))
		for _, issue := range issues {
			fmt.Println("  " + issue)
		}
		if db.IsFTSCorruptionOnly(issues) {
			fmt.Println("Corruption is limited to the search index. Run 'navidrome search rebuild' to fix it.")
		} else {
			fmt.Println("Corruption is not limited to the search index, and cannot be repaired automatically.")
			fmt.Println("Restore a backup (navidrome backup restore), or try SQLite's '.recover' command.")
		}
	}

	fmt.Println("Checking foreign keys...")
	violations, err := db.ForeignKeyCheck(ctx, database)
	switch {
	case err != nil:
		healthy = false
		fmt.Println("The foreign key check could not complete: " + err.Error())
	case len(violations) == 0:
		fmt.Println("Foreign key check passed.")
	default:
		healthy = false
		fmt.Printf("Foreign key check reported %d violation(s):\n", len(violations))
		for _, violation := range violations {
			fmt.Println("  " + violation)
		}
	}

	if !healthy {
		os.Exit(1)
	}
	fmt.Println("Database is healthy.")
}
