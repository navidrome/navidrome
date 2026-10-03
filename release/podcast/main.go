// release-podcast is a standalone, artifact-only release narration tool.
package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

//go:embed prompt.txt
var promptFiles embed.FS

const repository = "navidrome/navidrome"

type options struct {
	from, to, mode, textModel, ttsModel, voice, output, githubStage string
	allowPaid, dryRun, includePrereleases, force                    bool
}

func parseOptions(args []string, stderr io.Writer) (options, error) {
	var o options
	f := flag.NewFlagSet("release-podcast", flag.ContinueOnError)
	f.SetOutput(stderr)
	f.StringVar(&o.from, "from", "", "Published version to select, or inclusive first version of a range (v prefix optional)")
	f.StringVar(&o.to, "to", "", "Optional inclusive last published version of a range")
	f.StringVar(&o.mode, "mode", "validate", "validate (free), script, or audio")
	f.StringVar(&o.textModel, "text-model", os.Getenv("AUDIO_TEXT_MODEL"), "Reviewed script model; no default")
	f.StringVar(&o.ttsModel, "tts-model", os.Getenv("AUDIO_TTS_MODEL"), "Reviewed speech model; no default")
	f.StringVar(&o.voice, "voice", os.Getenv("AUDIO_VOICE"), "Compatible stock voice; no default")
	f.StringVar(&o.output, "output", "release-podcast", "Output directory (sources, manifest, script and MP3)")
	f.BoolVar(&o.dryRun, "dry-run", false, "Force validate mode; never contact OpenAI")
	f.BoolVar(&o.allowPaid, "allow-paid", false, "Explicitly permit bounded OpenAI requests for this local invocation")
	f.BoolVar(&o.includePrereleases, "include-prereleases", false, "Include published prereleases")
	f.BoolVar(&o.force, "force", false, "Permit another paid attempt and replacement of generated files")
	f.StringVar(&o.githubStage, "github-stage", "", "Actions entrypoint: prepare, script, or speech")
	if err := f.Parse(args); err != nil {
		return o, err
	}
	if f.NArg() != 0 {
		return o, errors.New("unexpected positional arguments")
	}
	if o.dryRun {
		o.mode = "validate"
	}
	if o.mode != "validate" && o.mode != "script" && o.mode != "audio" {
		return o, errors.New("mode must be validate, script, or audio")
	}
	if o.githubStage != "" && (o.from != "" || o.to != "" || o.allowPaid || o.dryRun || o.force || o.includePrereleases) {
		return o, errors.New("Actions stages use trusted event/environment configuration, not local selection flags")
	}
	return o, nil
}

type engine struct {
	client                                 *http.Client
	githubToken, openAIKey, output, prompt string
	command                                func(context.Context, string, ...string) ([]byte, error)
}

func newEngine(output string) (*engine, error) {
	abs, err := filepath.Abs(output)
	if err != nil {
		return nil, errors.New("invalid output directory")
	}
	prompt, err := promptFiles.ReadFile("prompt.txt")
	if err != nil {
		return nil, errors.New("embedded prompt unavailable")
	}
	return &engine{output: abs, prompt: string(prompt), githubToken: os.Getenv("GH_TOKEN"), openAIKey: os.Getenv("OPENAI_API_KEY"),
		client: &http.Client{Timeout: time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		command: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			if name != "git" && name != "ffprobe" && name != "ffmpeg" {
				return nil, errors.New("unsupported local tool")
			}
			cmd := exec.CommandContext(ctx, name, args...) // #nosec G204 -- fixed tool allowlist; arguments are data, never shell source.
			for _, item := range os.Environ() {
				if !strings.HasPrefix(item, "OPENAI_API_KEY=") && !strings.HasPrefix(item, "GH_TOKEN=") {
					cmd.Env = append(cmd.Env, item)
				}
			}
			return cmd.Output()
		},
	}, nil
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	o, err := parseOptions(args, stderr)
	if err != nil {
		return err
	}
	e, err := newEngine(o.output)
	if err != nil {
		return err
	}
	if o.githubStage != "" {
		return e.runGitHub(ctx, o.githubStage)
	}
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		return errors.New("Actions must use the guarded github-stage entrypoint")
	}
	return e.runLocal(ctx, o, stdout)
}

func (e *engine) runLocal(ctx context.Context, o options, stdout io.Writer) error {
	if o.mode != "validate" && (!o.allowPaid || e.openAIKey == "") {
		return errors.New("paid local modes require --allow-paid and OPENAI_API_KEY in the environment")
	}
	cfg, err := reviewedConfig(o.textModel, o.ttsModel, o.voice, o.mode)
	if err != nil {
		return err
	}
	if o.mode == "audio" {
		if err := e.mediaTools(ctx); err != nil {
			return err
		}
	}
	sources, err := e.resolveLocal(ctx, o)
	if err != nil {
		return err
	}
	m, err := e.prepare(ctx, sources, cfg, o.mode, "local", o.includePrereleases, o.force)
	if err != nil {
		return err
	}
	if o.mode == "validate" {
		if err := e.checkGeneratedFiles(); err != nil {
			return err
		}
		if err := e.saveSources(sources, m); err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, "Validated published sources; OpenAI requests: 0. Outputs:", e.output)
		return err
	}
	releaseLock, err := e.reserveLocal(m, o.force)
	if err != nil {
		return err
	}
	defer releaseLock()
	if o.force {
		for _, name := range []string{"transcript.txt", "release-podcast.mp3", "evidence.json"} {
			if err := os.Remove(filepath.Join(e.output, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
				return errors.New("cannot replace previous generated outputs")
			}
		}
	}
	if err := e.saveSources(sources, m); err != nil {
		return err
	}
	if err := e.generateScript(ctx); err != nil {
		return err
	}
	if o.mode == "audio" {
		if err := e.generateSpeech(ctx); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(stdout, "Generated review outputs:", e.output)
	return err
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, "Release podcast:", err)
		os.Exit(1)
	}
}
