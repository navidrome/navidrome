package cmd

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/spf13/cobra"
)

var pluginOutputFormat string

func init() {
	rootCmd.AddCommand(pluginRoot)

	pluginListCmd.Flags().StringVarP(&pluginOutputFormat, "format", "f", "table", "output format [supported values: table, csv, json]")
	pluginRoot.AddCommand(pluginListCmd)
}

var (
	pluginRoot = &cobra.Command{
		Use:   "plugin",
		Short: "Manage and inspect plugins",
		Long:  "List, inspect, enable, disable, configure, rescan, and validate plugins",
	}

	pluginListCmd = &cobra.Command{
		Use:   "list",
		Short: "List installed plugins",
		Run: func(cmd *cobra.Command, args []string) {
			runPluginList(cmd.Context())
		},
	}
)

// manifestSummary extracts the display name and version from a stored manifest JSON, falling
// back to the plugin ID when the manifest can't be parsed.
func manifestSummary(p model.Plugin) (name, version string) {
	var m struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal([]byte(p.Manifest), &m); err != nil {
		return p.ID, ""
	}
	return m.Name, m.Version
}

func formatPluginList(list model.Plugins, format string) (string, error) {
	switch format {
	case "json":
		b, err := json.MarshalIndent(list, "", "  ")
		if err != nil {
			return "", err
		}
		return string(b), nil
	case "csv":
		var sb strings.Builder
		w := csv.NewWriter(&sb)
		_ = w.Write([]string{"id", "name", "version", "enabled", "last error"})
		for _, p := range list {
			name, version := manifestSummary(p)
			_ = w.Write([]string{p.ID, name, version, fmt.Sprintf("%t", p.Enabled), p.LastError})
		}
		w.Flush()
		return sb.String(), w.Error()
	case "table":
		var sb strings.Builder
		w := tabwriter.NewWriter(&sb, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tVERSION\tENABLED\tLAST ERROR")
		for _, p := range list {
			name, version := manifestSummary(p)
			fmt.Fprintf(w, "%s\t%s\t%s\t%t\t%s\n", p.ID, name, version, p.Enabled, p.LastError)
		}
		w.Flush()
		return sb.String(), nil
	default:
		return "", fmt.Errorf("invalid output format %q (supported: table, csv, json)", format)
	}
}

func runPluginList(ctx context.Context) {
	requirePluginsEnabled(ctx)
	ds, ctx := getAdminContext(ctx)
	list, err := ds.Plugin(ctx).GetAll()
	if err != nil {
		log.Fatal(ctx, "Failed to list plugins", err)
	}
	out, err := formatPluginList(list, pluginOutputFormat)
	if err != nil {
		log.Fatal(ctx, "Failed to format output", err)
	}
	fmt.Print(out)
}

// requirePluginsEnabled aborts the command if the plugin system is disabled.
func requirePluginsEnabled(ctx context.Context) {
	if !conf.Server.Plugins.Enabled {
		log.Fatal(ctx, "Plugin system is disabled (set Plugins.Enabled to use this command)")
	}
}
