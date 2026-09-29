package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	format string
)

func init() {
	inspectCmd.Flags().StringVarP(&format, "format", "f", "jsonindent", "output format (pretty, toml, yaml, json, jsonindent)")
	rootCmd.AddCommand(inspectCmd)
}

var inspectCmd = &cobra.Command{
	Use:   "inspect [files to inspect]",
	Short: "Inspect tags",
	Long:  "Show file tags as seen by Navidrome",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runInspector(cmd.Context(), args)
	},
}

var marshalers = map[string]func(any) ([]byte, error){
	"pretty": prettyMarshal,
	"toml":   toml.Marshal,
	"yaml":   yaml.Marshal,
	"json":   json.Marshal,
	"jsonindent": func(v any) ([]byte, error) {
		return json.MarshalIndent(v, "", "  ")
	},
}

func prettyMarshal(v any) ([]byte, error) {
	out := v.([]core.InspectOutput)
	var res strings.Builder
	for i := range out {
		res.WriteString(fmt.Sprintf("====================\nFile: %s\n\n", out[i].File))
		t, _ := toml.Marshal(out[i].RawTags)
		res.WriteString(fmt.Sprintf("Raw tags:\n%s\n\n", t))
		t, _ = toml.Marshal(out[i].MappedTags)
		res.WriteString(fmt.Sprintf("Mapped tags:\n%s\n\n", t))
	}
	return []byte(res.String()), nil
}

func runInspector(ctx context.Context, args []string) {
	marshal := marshalers[format]
	if marshal == nil {
		log.Fatal("Invalid format", "format", format)
	}
	libs := loadLibraries(ctx)
	matcher := model.NewLibraryMatcher(libs)
	var out []core.InspectOutput
	for _, filePath := range args {
		if !model.IsAudioFile(filePath) {
			log.Warn("Not an audio file", "file", filePath)
			continue
		}
		lib, ok := libraryForFile(matcher, filePath)
		if !ok && len(libs) > 0 {
			log.Warn("File is not in any library, using the global PID config", "file", filePath)
		}
		output, err := core.Inspect(filePath, lib, "")
		if err != nil {
			log.Warn("Unable to process file", "file", filePath, "error", err)
			continue
		}

		out = append(out, *output)
	}
	data, _ := marshal(out)
	fmt.Println(string(data))
}

// loadLibraries reads the libraries, so each file gets its library's PID config. It never creates a DB.
func loadLibraries(ctx context.Context) model.Libraries {
	dbFile, _, _ := strings.Cut(conf.Server.DbPath, "?")
	if _, err := os.Stat(dbFile); err != nil {
		log.Warn(ctx, "No database found, using the global PID config", "path", dbFile)
		return nil
	}
	defer db.Init(ctx)()
	libs, err := persistence.New(db.Db()).Library().GetAll(ctx)
	if err != nil {
		log.Warn(ctx, "Could not load libraries, using the global PID config", err)
		return nil
	}
	for i := range libs {
		if absPath, err := filepath.Abs(libs[i].Path); err == nil {
			libs[i].Path = absPath
		}
	}
	return libs
}

// libraryForFile falls back to the default library with no overrides, which uses the global PID config.
func libraryForFile(matcher *model.LibraryMatcher, filePath string) (model.Library, bool) {
	if absPath, err := filepath.Abs(filePath); err == nil {
		if lib, ok := matcher.FindLibrary(absPath); ok {
			return lib, true
		}
	}
	return model.Library{ID: model.DefaultLibraryID}, false
}
