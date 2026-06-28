package cmd

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins"
	"github.com/spf13/cobra"
)

// pluginManager is the subset of *plugins.Manager the CLI needs.
type pluginManager interface {
	EnablePlugin(ctx context.Context, id string) error
	DisablePlugin(ctx context.Context, id string) error
	ValidatePluginConfig(ctx context.Context, id, configJSON string) error
	UpdatePluginConfig(ctx context.Context, id, configJSON string) error
	UpdatePluginUsers(ctx context.Context, id, usersJSON string, allUsers bool) error
	UpdatePluginLibraries(ctx context.Context, id, librariesJSON string, allLibraries, allowWriteAccess bool) error
	RescanPlugins(ctx context.Context) error
}

var pluginOutputFormat string

var (
	editConfig      string
	editConfigFile  string
	editUsers       string
	editAllUsers    bool
	editLibraries   string
	editAllLibs     bool
	editWriteAccess bool
	editNoWrite     bool
)

func init() {
	rootCmd.AddCommand(pluginRoot)

	pluginListCmd.Flags().StringVarP(&pluginOutputFormat, "format", "f", "table", "output format [supported values: table, csv, json]")
	pluginRoot.AddCommand(pluginListCmd)
	pluginRoot.AddCommand(pluginEnableCmd)
	pluginRoot.AddCommand(pluginDisableCmd)

	pluginEditCmd.Flags().StringVar(&editConfig, "config", "", "plugin config as JSON")
	pluginEditCmd.Flags().StringVar(&editConfigFile, "config-file", "", "read plugin config JSON from a file ('-' for stdin)")
	pluginEditCmd.MarkFlagsMutuallyExclusive("config", "config-file")
	pluginEditCmd.Flags().StringVar(&editUsers, "users", "", "comma-separated usernames the plugin may access")
	pluginEditCmd.Flags().BoolVar(&editAllUsers, "all-users", false, "grant the plugin access to all users")
	pluginEditCmd.MarkFlagsMutuallyExclusive("users", "all-users")
	pluginEditCmd.Flags().StringVar(&editLibraries, "libraries", "", "comma-separated library IDs the plugin may access")
	pluginEditCmd.Flags().BoolVar(&editAllLibs, "all-libraries", false, "grant the plugin access to all libraries")
	pluginEditCmd.MarkFlagsMutuallyExclusive("libraries", "all-libraries")
	pluginEditCmd.Flags().BoolVar(&editWriteAccess, "write-access", false, "allow the plugin write access to libraries")
	pluginEditCmd.Flags().BoolVar(&editNoWrite, "no-write-access", false, "deny the plugin write access to libraries")
	pluginEditCmd.MarkFlagsMutuallyExclusive("write-access", "no-write-access")
	pluginRoot.AddCommand(pluginEditCmd)

	pluginInfoCmd.Flags().StringVarP(&pluginOutputFormat, "format", "f", "text", "output format [supported values: text, json]")
	pluginRoot.AddCommand(pluginInfoCmd)
	pluginRoot.AddCommand(pluginValidateCmd)
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

	pluginEnableCmd = &cobra.Command{
		Use:   "enable <id>",
		Short: "Enable a plugin",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			requirePluginsEnabled(cmd.Context())
			_, ctx := getAdminContext(cmd.Context())
			mgr := GetPluginManager(ctx)
			if err := enablePlugin(ctx, mgr, args[0]); err != nil {
				log.Fatal(ctx, "Failed to enable plugin", "id", args[0], err)
			}
		},
	}

	pluginDisableCmd = &cobra.Command{
		Use:   "disable <id>",
		Short: "Disable a plugin",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			requirePluginsEnabled(cmd.Context())
			_, ctx := getAdminContext(cmd.Context())
			mgr := GetPluginManager(ctx)
			if err := disablePlugin(ctx, mgr, args[0]); err != nil {
				log.Fatal(ctx, "Failed to disable plugin", "id", args[0], err)
			}
		},
	}
)

var (
	pluginInfoCmd = &cobra.Command{
		Use:   "info <id|file.ndp>",
		Short: "Show details for an installed plugin or a .ndp package",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runPluginInfo(cmd.Context(), args[0])
		},
	}

	pluginValidateCmd = &cobra.Command{
		Use:   "validate <id|file.ndp>",
		Short: "Validate an installed plugin or a .ndp package manifest",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runPluginValidate(cmd.Context(), args[0])
		},
	}
)

func isPackagePath(arg string) bool {
	if !strings.HasSuffix(arg, plugins.PackageExtension) {
		return false
	}
	info, err := os.Stat(arg)
	return err == nil && !info.IsDir()
}

func formatPluginInfo(p *model.Plugin, format string) (string, error) {
	if format == "json" {
		b, err := json.MarshalIndent(p, "", "  ")
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	name, version := manifestSummary(*p)
	var sb strings.Builder
	fmt.Fprintf(&sb, "ID:          %s\n", p.ID)
	fmt.Fprintf(&sb, "Name:        %s\n", name)
	fmt.Fprintf(&sb, "Version:     %s\n", version)
	fmt.Fprintf(&sb, "Enabled:     %t\n", p.Enabled)
	fmt.Fprintf(&sb, "Path:        %s\n", p.Path)
	fmt.Fprintf(&sb, "SHA256:      %s\n", p.SHA256)
	fmt.Fprintf(&sb, "All users:   %t\n", p.AllUsers)
	fmt.Fprintf(&sb, "All libs:    %t\n", p.AllLibraries)
	fmt.Fprintf(&sb, "Write access:%t\n", p.AllowWriteAccess)
	if p.Config != "" {
		fmt.Fprintf(&sb, "Config:      %s\n", p.Config)
	}
	if p.LastError != "" {
		fmt.Fprintf(&sb, "Last error:  %s\n", p.LastError)
	}
	return sb.String(), nil
}

func formatManifestInfo(m *plugins.Manifest, format string) (string, error) {
	if format == "json" {
		b, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Name:    %s\n", m.Name)
	fmt.Fprintf(&sb, "Version: %s\n", m.Version)
	fmt.Fprintf(&sb, "Author:  %s\n", m.Author)
	if m.Description != nil {
		fmt.Fprintf(&sb, "Description: %s\n", *m.Description)
	}
	if m.Website != nil {
		fmt.Fprintf(&sb, "Website: %s\n", *m.Website)
	}
	return sb.String(), nil
}

func runPluginInfo(ctx context.Context, arg string) {
	if isPackagePath(arg) {
		m, err := plugins.ReadPackageManifest(arg)
		if err != nil {
			log.Fatal(ctx, "Failed to read package", "path", arg, err)
		}
		out, err := formatManifestInfo(m, pluginOutputFormat)
		if err != nil {
			log.Fatal(ctx, "Failed to format output", err)
		}
		fmt.Print(out)
		return
	}
	requirePluginsEnabled(ctx)
	ds, ctx := getAdminContext(ctx)
	p, err := ds.Plugin(ctx).Get(arg)
	if err != nil {
		log.Fatal(ctx, "Plugin not found", "id", arg, err)
	}
	out, err := formatPluginInfo(p, pluginOutputFormat)
	if err != nil {
		log.Fatal(ctx, "Failed to format output", err)
	}
	fmt.Print(out)
}

func runPluginValidate(ctx context.Context, arg string) {
	if isPackagePath(arg) {
		if _, err := plugins.ValidatePackage(arg); err != nil {
			log.Fatal(ctx, "Validation failed", "path", arg, err)
		}
		fmt.Printf("%s: OK\n", arg)
		return
	}
	requirePluginsEnabled(ctx)
	ds, ctx := getAdminContext(ctx)
	p, err := ds.Plugin(ctx).Get(arg)
	if err != nil {
		log.Fatal(ctx, "Plugin not found", "id", arg, err)
	}
	if _, err := plugins.ParseManifest([]byte(p.Manifest)); err != nil {
		log.Fatal(ctx, "Validation failed", "id", arg, err)
	}
	fmt.Printf("%s: OK\n", arg)
}

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

func enablePlugin(ctx context.Context, mgr pluginManager, id string) error {
	return mgr.EnablePlugin(ctx, id)
}

func disablePlugin(ctx context.Context, mgr pluginManager, id string) error {
	return mgr.DisablePlugin(ctx, id)
}

type pluginEditOptions struct {
	config       *string // nil = leave unchanged
	users        *string
	allUsers     *bool
	libraries    *string
	allLibraries *bool
	writeAccess  *bool
}

var pluginEditCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Update a plugin's config and/or permissions",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		requirePluginsEnabled(cmd.Context())
		_, ctx := getAdminContext(cmd.Context())
		mgr := GetPluginManager(ctx)
		opts := buildEditOptionsFromFlags(cmd)
		if err := applyPluginEdit(ctx, mgr, args[0], opts); err != nil {
			log.Fatal(ctx, "Failed to edit plugin", "id", args[0], err)
		}
	},
}

func buildEditOptionsFromFlags(cmd *cobra.Command) pluginEditOptions {
	var opts pluginEditOptions
	switch {
	case cmd.Flags().Changed("config"):
		c := editConfig
		opts.config = &c
	case cmd.Flags().Changed("config-file"):
		c := readConfigFile(editConfigFile)
		opts.config = &c
	}
	if cmd.Flags().Changed("users") {
		u := editUsers
		opts.users = &u
	}
	if cmd.Flags().Changed("all-users") {
		opts.allUsers = &editAllUsers
	}
	if cmd.Flags().Changed("libraries") {
		l := editLibraries
		opts.libraries = &l
	}
	if cmd.Flags().Changed("all-libraries") {
		opts.allLibraries = &editAllLibs
	}
	if cmd.Flags().Changed("write-access") || cmd.Flags().Changed("no-write-access") {
		wa := editWriteAccess && !editNoWrite
		opts.writeAccess = &wa
	}
	return opts
}

func readConfigFile(path string) string {
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		log.Fatal("Failed to read config file", "path", path, err)
	}
	return string(data)
}

func applyPluginEdit(ctx context.Context, mgr pluginManager, id string, opts pluginEditOptions) error {
	if opts.config == nil && opts.users == nil && opts.allUsers == nil &&
		opts.libraries == nil && opts.allLibraries == nil && opts.writeAccess == nil {
		return fmt.Errorf("nothing to update: provide at least one of --config/--users/--libraries/--write-access")
	}
	if opts.config != nil {
		if err := mgr.ValidatePluginConfig(ctx, id, *opts.config); err != nil {
			return fmt.Errorf("invalid config: %w", err)
		}
		if err := mgr.UpdatePluginConfig(ctx, id, *opts.config); err != nil {
			return err
		}
	}
	if opts.users != nil || opts.allUsers != nil {
		users := ""
		if opts.users != nil {
			users = *opts.users
		}
		allUsers := opts.allUsers != nil && *opts.allUsers
		if err := mgr.UpdatePluginUsers(ctx, id, users, allUsers); err != nil {
			return err
		}
	}
	if opts.libraries != nil || opts.allLibraries != nil || opts.writeAccess != nil {
		libs := ""
		if opts.libraries != nil {
			libs = *opts.libraries
		}
		allLibs := opts.allLibraries != nil && *opts.allLibraries
		writeAccess := opts.writeAccess != nil && *opts.writeAccess
		if err := mgr.UpdatePluginLibraries(ctx, id, libs, allLibs, writeAccess); err != nil {
			return err
		}
	}
	return nil
}
