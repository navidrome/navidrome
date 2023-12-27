package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/scanner/metadata"
	"github.com/navidrome/navidrome/tests"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	extractor string
	format    string
)

func init() {
	inspectCmd.Flags().StringVarP(&extractor, "extractor", "x", "", "extractor to use (ffmpeg or taglib, default: auto)")
	inspectCmd.Flags().StringVarP(&format, "format", "f", "pretty", "output format (pretty, toml, yaml, json, jsonindent)")
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
	RawTags    metadata.ParsedTags
	MappedTags model.MediaFile
}

func runInspector(args []string) {
	if extractor != "" {
		conf.Server.Scanner.Extractor = extractor
	}
	log.Info("Using extractor", "extractor", conf.Server.Scanner.Extractor)
	md, err := metadata.Extract(args...)
	if err != nil {
		log.Fatal("Error extracting tags", err)
	}
	mapper := scanner.NewMediaFileMapper(conf.Server.MusicFolder, &tests.MockedGenreRepo{})
	marshal := marshalers[format]
	if marshal == nil {
		log.Fatal("Invalid format", "format", format)
	}
	var out []inspectorOutput
	for k, v := range md {
		if !model.IsAudioFile(k) {
			continue
		}
		if len(v.Tags) == 0 {
			continue
		}
		out = append(out, inspectorOutput{
			File:       k,
			RawTags:    v.Tags,
			MappedTags: mapper.ToMediaFile(v),
		})
	}
	data, _ := marshal(out)
	fmt.Println(string(data))
}
