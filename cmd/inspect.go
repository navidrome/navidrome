package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/core/storage"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/metadata"
	"github.com/navidrome/navidrome/scanner/metadata_old"
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
		runInspector(args)
	},
}

var marshalers = map[string]func(interface{}) ([]byte, error){
	"pretty": prettyMarshal,
	"toml":   toml.Marshal,
	"yaml":   yaml.Marshal,
	"json":   json.Marshal,
	"jsonindent": func(v interface{}) ([]byte, error) {
		return json.MarshalIndent(v, "", "  ")
	},
}

func prettyMarshal(v interface{}) ([]byte, error) {
	out := v.([]inspectorOutput)
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

type inspectorOutput struct {
	File       string
	RawTags    metadata_old.ParsedTags
	MappedTags model.MediaFile
}

func runInspector(args []string) {
	marshal := marshalers[format]
	if marshal == nil {
		log.Fatal("Invalid format", "format", format)
	}
	var out []inspectorOutput
	for _, filePath := range args {
		if !model.IsAudioFile(filePath) {
			log.Warn("Not an audio file", "file", filePath)
			continue
		}
		path, file := filepath.Split(filePath)
		s, err := storage.For(path)
		if err != nil {
			log.Fatal("Error creating storage", err)
		}
		fs, err := s.FS()
		if err != nil {
			log.Fatal("Error creating FS", err)
		}
		tags, err := fs.ReadTags(file)
		if err != nil {
			log.Warn("Error reading tags", "file", file, "err", err)
			continue
		}
		if len(tags[file].Tags) == 0 {
			log.Warn("No tags found", "file", file)
			continue
		}
		md := metadata.New(file, tags[file])
		out = append(out, inspectorOutput{
			File:       file,
			RawTags:    tags[file].Tags,
			MappedTags: md.ToMediaFile(1, ""),
		})
	}
	data, _ := marshal(out)
	fmt.Println(string(data))
}
