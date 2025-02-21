package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
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

func runInspector(args []string) {
	marshal := marshalers[format]
	if marshal == nil {
		log.Fatal("Invalid format", "format", format)
	}
	var out []core.InspectOutput
	for _, filePath := range args {
		if !model.IsAudioFile(filePath) {
			log.Warn("Not an audio file", "file", filePath)
			continue
		}
		output, err := core.Inspect(filePath, 1, "")
		if err != nil {
			log.Warn("Unable to process file", "file", filePath, "error", err)
			continue
		}

		out = append(out, *output)
	}
	data, _ := marshal(out)
	fmt.Println(string(data))
}
