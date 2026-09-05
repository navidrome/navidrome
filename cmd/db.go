package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"io"
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
			"report what was found, including whether 'navidrome search rebuild' can fix it. " +
			"This command never alters your data",
		Run: func(cmd *cobra.Command, _ []string) {
			runDoctor(cmd.Context())
		},
	}
)

func runDoctor(ctx context.Context) {
	requireExistingDB()

	healthy := doctor(ctx, db.Db(), os.Stdout)
	db.Close(ctx)
	if !healthy {
		os.Exit(1)
	}
}

const recoveryAdvice = "Restore a backup (navidrome backup restore), or try SQLite's '.recover' command."

func doctor(ctx context.Context, database *sql.DB, out io.Writer) bool {
	printFindings := func(check, noun string, items []string) {
		fmt.Fprintf(out, "%s reported %d %s:\n", check, len(items), noun)
		for _, item := range items {
			fmt.Fprintln(out, "  "+item)
		}
	}

	healthy := true

	fmt.Fprintln(out, "Checking database integrity...")
	issues, err := db.IntegrityCheck(ctx, database)
	switch {
	case err != nil:
		fmt.Fprintln(out, "The integrity check could not complete: "+err.Error())
		fmt.Fprintln(out, recoveryAdvice)
		return false
	case len(issues) == 0:
		fmt.Fprintln(out, "Integrity check passed.")
	default:
		healthy = false
		printFindings("Integrity check", "issue(s)", issues)
		if db.IsFTSCorruptionOnly(issues) {
			fmt.Fprintln(out, "Corruption is limited to the search index. Run 'navidrome search rebuild' to fix it.")
		} else {
			fmt.Fprintln(out, "Corruption is not limited to the search index, and cannot be repaired automatically.")
			fmt.Fprintln(out, recoveryAdvice)
		}
	}

	fmt.Fprintln(out, "Checking foreign keys...")
	violations, err := db.ForeignKeyCheck(ctx, database)
	switch {
	case err != nil:
		healthy = false
		fmt.Fprintln(out, "The foreign key check could not complete: "+err.Error())
	case len(violations) == 0:
		fmt.Fprintln(out, "Foreign key check passed.")
	default:
		healthy = false
		printFindings("Foreign key check", "violation(s)", violations)
	}

	if healthy {
		fmt.Fprintln(out, "Database is healthy.")
	}
	return healthy
}
