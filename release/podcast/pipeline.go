package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	maxPromptBytes     = 65536
	maxOutputTokens    = 3000
	maxScriptChars     = 2500
	miniInputBytes     = 2000
	modeledAudioTokens = 6000
	maxCostUSD         = 0.10
	speechInstructions = "Speak in clear, calm English with a warm, measured delivery. Read API as letters."
	intro              = "Welcome to the Navidrome release recap. This narration uses an AI-generated voice."
	closing            = "For the full details and upgrade guidance, read the official release notes linked alongside this transcript."
)

type modelConfig struct {
	TextModel    string  `json:"text_model"`
	TTSModel     string  `json:"tts_model"`
	Voice        string  `json:"voice"`
	Speed        float64 `json:"speed"`
	Instructions string  `json:"speech_instructions"`
}

var textRates = map[string][2]float64{"gpt-6-luna": {0.10, 0.50}, "gpt-4.1-mini-2025-04-14": {0.40, 1.60}}
var speechRates = map[string]float64{"tts-1": 15, "tts-1-hd": 30, "gpt-4o-mini-tts-2025-12-15": 0}

func reviewedConfig(text, tts, voice, mode string) (modelConfig, error) {
	c := modelConfig{TextModel: text, TTSModel: tts, Voice: voice, Speed: 1}
	if _, ok := textRates[text]; !ok && (mode != "validate" || text != "") {
		return c, errors.New("select a reviewed text model; no default is selected")
	}
	if _, ok := speechRates[tts]; !ok && (mode == "audio" || tts != "") {
		return c, errors.New("select a reviewed speech model")
	}
	voices := ",alloy,echo,fable,onyx,nova,shimmer,"
	if tts == "gpt-4o-mini-tts-2025-12-15" {
		voices += "ash,ballad,coral,sage,verse,marin,cedar,"
		c.Instructions = speechInstructions
	}
	if (voice != "" || mode == "audio") && (voice == "" || !slices.Contains(strings.Split(strings.Trim(voices, ","), ","), voice)) {
		return c, errors.New("select a compatible reviewed stock voice")
	}
	return c, nil
}

func narrationByteLimit(c modelConfig) int {
	if c.Instructions != "" {
		return miniInputBytes - len(c.Instructions)
	}
	return maxScriptChars * 4 // Character-priced models; Unicode still checked separately.
}

type manifest struct {
	Version             int             `json:"version"`
	Mode                string          `json:"mode"`
	Origin              string          `json:"origin"`
	Config              modelConfig     `json:"config"`
	Cost                *float64        `json:"modeled_cost_usd"`
	SourceHash          string          `json:"source_sha256"`
	PromptHash          string          `json:"prompt_sha256"`
	GenerationHash      string          `json:"generation_sha256"`
	ImplementationSHA   string          `json:"implementation_sha"`
	EventSHA            string          `json:"event_sha,omitempty"`
	RunID               string          `json:"run_id,omitempty"`
	RunAttempt          string          `json:"run_attempt,omitempty"`
	CreatedAt           string          `json:"created_at"`
	Reservation         string          `json:"reservation"`
	TextRequests        int             `json:"text_requests"`
	SpeechRequests      int             `json:"speech_requests"`
	IncludePrereleases  bool            `json:"include_prereleases"`
	Force               bool            `json:"force_regenerate"`
	Status              string          `json:"status"`
	ScriptHash          string          `json:"script_sha256,omitempty"`
	AudioHash           string          `json:"audio_sha256,omitempty"`
	Words               int             `json:"word_count,omitempty"`
	Characters          int             `json:"character_count,omitempty"`
	Duration            float64         `json:"duration_seconds,omitempty"`
	DurationNeedsReview bool            `json:"duration_needs_review,omitempty"`
	TextUsage           json.RawMessage `json:"text_usage,omitempty"`
}

func hashBytes(data []byte) string { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }
func hashJSON(value any) string    { data, _ := json.Marshal(value); return hashBytes(data) }

func (e *engine) textPayload(sources []source, c modelConfig) map[string]any {
	input, _ := json.Marshal(map[string]any{"sources": promptSources(sources), "required_cautions": cautions(sources),
		"introduction": intro, "closing": closing, "narration_limits": map[string]int{"max_words": 280, "max_characters": maxScriptChars, "max_utf8_bytes": narrationByteLimit(c)}})
	p := map[string]any{"model": c.TextModel, "store": false, "max_output_tokens": maxOutputTokens, "instructions": e.prompt, "input": string(input),
		"text": map[string]any{"format": map[string]any{"type": "json_schema", "name": "release_narration", "strict": true, "schema": narrationSchema()}}}
	if c.TextModel == "gpt-6-luna" {
		p["reasoning"] = map[string]string{"effort": "low"}
	}
	return p
}

func modeledCost(payload any, c modelConfig, mode string) (*float64, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.New("cannot encode prompt")
	}
	if len(data) > maxPromptBytes {
		return nil, errors.New("complete prompt exceeds byte limit; never truncate warnings")
	}
	rate, ok := textRates[c.TextModel]
	if !ok {
		return nil, nil
	}
	// Conservative byte-level input-token count, plus protocol framing.
	cost := (float64(len(data)+1024)*rate[0] + maxOutputTokens*rate[1]) / 1e6
	if mode == "audio" {
		if c.Instructions != "" {
			cost += (float64(miniInputBytes)*0.60 + modeledAudioTokens*12) / 1e6
		} else {
			cost += maxScriptChars * speechRates[c.TTSModel] / 1e6
		}
	}
	if cost > maxCostUSD {
		return nil, errors.New("modeled cost exceeds $0.10; review source size or models")
	}
	return &cost, nil
}

func (e *engine) prepare(ctx context.Context, sources []source, c modelConfig, mode, origin string, allow, force bool) (manifest, error) {
	var m manifest
	seen := map[int64]bool{}
	total := 0
	ids := []int64{}
	if len(sources) < 1 || len(sources) > 3 {
		return m, errors.New("select one to three releases")
	}
	for _, s := range sources {
		if seen[s.ID] {
			return m, errors.New("duplicate release IDs")
		}
		seen[s.ID] = true
		ids = append(ids, s.ID)
		total += len(s.Body)
	}
	if total > maxSourceBytes {
		return m, errors.New("combined notes exceed byte limit")
	}
	cost, err := modeledCost(e.textPayload(sources, c), c, mode)
	if err != nil {
		return m, err
	}
	sha, err := e.command(ctx, "git", "rev-parse", "HEAD")
	if err != nil {
		sha = []byte("unknown")
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	m = manifest{Version: 2, Mode: mode, Origin: origin, Config: c, Cost: cost, SourceHash: hashJSON(sources), PromptHash: hashBytes([]byte(e.prompt)),
		ImplementationSHA: strings.TrimSpace(string(sha)), CreatedAt: time.Now().UTC().Format(time.RFC3339), IncludePrereleases: allow, Force: force, Status: "prepared",
		Reservation: "release-podcast-attempt-" + hashJSON(map[string]any{"repo": repository, "ids": ids, "mode": mode})}
	m.GenerationHash = hashJSON(map[string]any{"sources": sources, "config": c, "prompt": e.prompt, "implementation": m.ImplementationSHA})
	return m, nil
}

func (e *engine) writeFile(name string, data []byte) error {
	if err := os.MkdirAll(e.output, 0750); err != nil {
		return errors.New("cannot create output directory")
	}
	f, err := os.CreateTemp(e.output, ".release-podcast-*")
	if err != nil {
		return errors.New("cannot create output file")
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(data); err != nil {
		f.Close()
		return errors.New("cannot write output")
	}
	if err := f.Close(); err != nil {
		return errors.New("cannot close output")
	}
	if err := os.Rename(f.Name(), filepath.Join(e.output, name)); err != nil {
		return errors.New("cannot install output")
	}
	return nil
}

func (e *engine) writeJSON(name string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return errors.New("cannot encode output")
	}
	return e.writeFile(name, append(data, '\n'))
}

func (e *engine) readJSON(name string, target any) error {
	data, err := os.ReadFile(filepath.Join(e.output, name))
	if err != nil {
		return errors.New("required output checkpoint is unavailable")
	}
	if len(data) > 8<<20 || json.Unmarshal(data, target) != nil {
		return errors.New("invalid output checkpoint")
	}
	return nil
}

func (e *engine) saveSources(sources []source, m manifest) error {
	if err := e.writeJSON("sources.json", sources); err != nil {
		return err
	}
	return e.writeJSON("manifest.json", m)
}

func (e *engine) reserveLocal(m manifest, force bool) (func(), error) {
	if err := os.MkdirAll(e.output, 0750); err != nil {
		return nil, errors.New("cannot create output directory")
	}
	lockPath := filepath.Join(e.output, ".release-podcast.lock")
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, errors.New("output directory is locked by another attempt; review a stale lock before removal")
	}
	lock.Close()
	release := func() { _ = os.Remove(lockPath) }
	fail := func(err error) (func(), error) { release(); return nil, err }
	ledger := filepath.Join(e.output, ".attempts")
	if err := os.MkdirAll(ledger, 0750); err != nil {
		return fail(errors.New("cannot create local attempt ledger"))
	}
	attempt := filepath.Join(ledger, m.Reservation+".json")
	if !force {
		if err := e.checkGeneratedFiles(); err != nil {
			return fail(err)
		}
	}
	flags := os.O_CREATE | os.O_EXCL | os.O_WRONLY
	if force {
		flags = os.O_CREATE | os.O_TRUNC | os.O_WRONLY
	}
	f, err := os.OpenFile(attempt, flags, 0600)
	if err != nil {
		return fail(errors.New("paid attempt already reserved or ledger unavailable; explicit --force permits another charge"))
	}
	data, _ := json.Marshal(m)
	_, writeErr := f.Write(data)
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil {
		return fail(errors.New("cannot persist attempt reservation; no paid request made"))
	}
	return release, nil
}

func (e *engine) checkGeneratedFiles() error {
	for _, name := range []string{"transcript.txt", "release-podcast.mp3", "evidence.json"} {
		_, err := os.Lstat(filepath.Join(e.output, name))
		if err == nil {
			return errors.New("generated files already exist; use a fresh output directory or explicit --force in a paid mode")
		}
		if !errors.Is(err, os.ErrNotExist) {
			return errors.New("cannot inspect existing output files")
		}
	}
	return nil
}

func (e *engine) stage(ctx context.Context, kind string) (manifest, []source, error) {
	var m manifest
	var sources []source
	if err := e.readJSON("manifest.json", &m); err != nil {
		return m, nil, err
	}
	if err := e.readJSON("sources.json", &sources); err != nil {
		return m, nil, err
	}
	if e.openAIKey == "" || m.Mode == "validate" {
		return m, nil, errors.New("paid stage is not authorized or key is missing")
	}
	if (m.Origin != "local" && m.Origin != "github") || (os.Getenv("GITHUB_ACTIONS") == "true") != (m.Origin == "github") {
		return m, nil, errors.New("checkpoint origin does not match the execution context")
	}
	if m.Origin == "github" {
		if _, err := githubEvent(); err != nil {
			return m, nil, err
		}
		if os.Getenv("AUDIO_ENABLED") != "true" {
			return m, nil, errors.New("Actions paid generation is disabled")
		}
		if m.RunID != os.Getenv("GITHUB_RUN_ID") || m.RunAttempt != os.Getenv("GITHUB_RUN_ATTEMPT") {
			return m, nil, errors.New("checkpoint belongs to another Actions attempt")
		}
		c, err := reviewedConfig(os.Getenv("AUDIO_TEXT_MODEL"), os.Getenv("AUDIO_TTS_MODEL"), os.Getenv("AUDIO_VOICE"), m.Mode)
		if err != nil || c != m.Config {
			return m, nil, errors.New("configuration changed after preparation")
		}
	}
	if (kind == "text" && (m.TextRequests != 0 || m.Status != "prepared")) || (kind == "speech" && (m.SpeechRequests != 0 || m.Status != "script_validated" || m.Mode != "audio")) {
		return m, nil, errors.New("stage already attempted or checkpoint not ready")
	}
	if hashJSON(sources) != m.SourceHash {
		return m, nil, errors.New("source checkpoint changed")
	}
	if err := e.recheck(ctx, m, sources); err != nil {
		return m, nil, err
	}
	return m, sources, nil
}

func (e *engine) generateScript(ctx context.Context) error {
	m, sources, err := e.stage(ctx, "text")
	if err != nil {
		return err
	}
	payload := e.textPayload(sources, m.Config)
	if _, err := modeledCost(payload, m.Config, m.Mode); err != nil {
		return err
	}
	m.TextRequests = 1
	m.Status = "script_requested"
	if err := e.writeJSON("manifest.json", m); err != nil {
		return err
	}
	data, ct, err := e.request(ctx, "https://api.openai.com/v1/responses", e.openAIKey, payload, 1<<20)
	if err != nil {
		return err
	}
	if ct != "application/json" {
		return errors.New("text API returned an unexpected content type")
	}
	result, usage, err := parseResponse(data)
	if err != nil {
		return err
	}
	text, err := validateNarration(result, sources)
	if err != nil {
		return err
	}
	if len(strings.TrimSuffix(text, "\n")) > narrationByteLimit(m.Config) {
		return errors.New("narration exceeds configured speech input limit; shorten optional highlights")
	}
	if err := e.writeFile("transcript.txt", []byte(text)); err != nil {
		return err
	}
	if err := e.writeJSON("evidence.json", result); err != nil {
		return err
	}
	m.Status = "script_validated"
	m.ScriptHash = hashBytes([]byte(text))
	m.Words = len(strings.Fields(text))
	m.Characters = len([]rune(strings.TrimSuffix(text, "\n")))
	m.TextUsage = usage
	return e.writeJSON("manifest.json", m)
}

func (e *engine) mediaTools(ctx context.Context) error {
	for _, tool := range []string{"ffprobe", "ffmpeg"} {
		toolCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, err := e.command(toolCtx, tool, "-version")
		cancel()
		if err != nil {
			return fmt.Errorf("%s is required before paid audio generation", tool)
		}
	}
	return nil
}

func (e *engine) generateSpeech(ctx context.Context) error {
	m, sources, err := e.stage(ctx, "speech")
	if err != nil {
		return err
	}
	text, err := os.ReadFile(filepath.Join(e.output, "transcript.txt"))
	if err != nil {
		return errors.New("script checkpoint unavailable")
	}
	var result narration
	if err := e.readJSON("evidence.json", &result); err != nil {
		return err
	}
	validated, err := validateNarration(result, sources)
	if err != nil || validated != string(text) || hashBytes(text) != m.ScriptHash {
		return errors.New("script checkpoint changed or is invalid")
	}
	if err := e.mediaTools(ctx); err != nil {
		return err
	}
	input := strings.TrimSuffix(string(text), "\n")
	payload := map[string]any{"model": m.Config.TTSModel, "voice": m.Config.Voice, "input": input, "response_format": "mp3", "speed": m.Config.Speed}
	if m.Config.Instructions != "" {
		if len(input)+len(m.Config.Instructions) > miniInputBytes {
			return errors.New("mini TTS conservative input-token limit exceeded")
		}
		payload["instructions"] = m.Config.Instructions
	}
	m.SpeechRequests = 1
	m.Status = "speech_requested"
	if err := e.writeJSON("manifest.json", m); err != nil {
		return err
	}
	data, ct, err := e.request(ctx, "https://api.openai.com/v1/audio/speech", e.openAIKey, payload, 10<<20)
	if err != nil {
		return err
	}
	if (ct != "audio/mpeg" && ct != "audio/mp3" && ct != "application/octet-stream") || len(data) == 0 {
		return errors.New("speech API did not return audio")
	}
	if err := e.writeFile("audio.tmp", data); err != nil {
		return err
	}
	temporary := filepath.Join(e.output, "audio.tmp")
	defer os.Remove(temporary)
	mediaCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	probe, err := e.command(mediaCtx, "ffprobe", "-v", "error", "-f", "mp3", "-protocol_whitelist", "file,pipe", "-show_entries", "format=duration:stream=codec_name", "-of", "json", temporary)
	if err != nil {
		return errors.New("speech response cannot be inspected as MP3")
	}
	var info struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
		Streams []struct {
			Codec string `json:"codec_name"`
		} `json:"streams"`
	}
	if json.Unmarshal(probe, &info) != nil || len(info.Streams) == 0 {
		return errors.New("invalid MP3 metadata")
	}
	duration, err := strconv.ParseFloat(info.Format.Duration, 64)
	if err != nil || math.IsNaN(duration) || math.IsInf(duration, 0) || duration <= 0 {
		return errors.New("invalid MP3 duration")
	}
	for _, stream := range info.Streams {
		if stream.Codec != "mp3" {
			return errors.New("speech response is not MP3")
		}
	}
	if _, err := e.command(mediaCtx, "ffmpeg", "-v", "error", "-xerror", "-f", "mp3", "-protocol_whitelist", "file,pipe", "-i", temporary, "-f", "null", "-"); err != nil {
		return errors.New("MP3 failed complete decoding")
	}
	if err := os.Rename(temporary, filepath.Join(e.output, "release-podcast.mp3")); err != nil {
		return errors.New("cannot install validated MP3")
	}
	m.Status = "audio_validated"
	m.Duration = duration
	m.DurationNeedsReview = duration < 105 || duration > 145
	m.AudioHash = hashBytes(data)
	return e.writeJSON("manifest.json", m)
}

func strictDecode(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return errors.New("invalid structured narration schema")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errors.New("unexpected data after narration")
	}
	return nil
}
