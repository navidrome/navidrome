package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func fixtures(t *testing.T) ([]releaseRecord, []source) {
	t.Helper()
	data, err := os.ReadFile("testdata/releases.json")
	if err != nil {
		t.Fatal(err)
	}
	var records []releaseRecord
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatal(err)
	}
	var sources []source
	for _, r := range records {
		s, err := normalizeRelease(r, false)
		if err != nil {
			t.Fatal(err)
		}
		sources = append(sources, s)
	}
	return records, sources
}

func exampleNarration(t *testing.T, sources []source) narration {
	t.Helper()
	texts := []struct {
		index       int
		text, quote string
	}{
		{0, "Version 0.64.0 introduced experimental Jellyfin music support, which must be explicitly enabled.", "Enable it with `Jellyfin.Enabled = true`."},
		{0, "Back up your database before upgrading because internal IDs change; clients may need to resync cached IDs.", "back up your database before upgrading"},
		{0, "Plugin authors must migrate to the host HTTP service and review restrictions on private or loopback network addresses.", "Plugins must use the host HTTP service instead"},
		{0, "Shares now belong to their creator, and admins cannot create them for another user.", "Shares are always owned by the user who creates them."},
		{0, "Negative configuration durations are rejected at startup, and unknown options produce warnings.", "Negative values are rejected at startup."},
		{0, "Security fixes protect library access and plugin networking, alongside improvements to artwork, sorting and playlist imports.", "This release fixes several reported vulnerabilities."},
		{0, "Database restore also avoids wiping existing data when the backup file is missing.", "wiping the database when the backup file does not exist"},
		{1, "Version 0.64.1 is a security release fixing five vulnerabilities; upgrade as soon as practical.", "This is a security release. It fixes five vulnerabilities"},
		{1, "Jellyfin client compatibility improves, and Quick Connect makes signing in easier.", "Add Quick Connect sign-in."},
		{1, "Local discovery is opt-in, and Docker users need host networking for discovery broadcasts.", "Docker users need host networking for the UDP broadcast to reach the container."},
		{1, "Smart playlists can reference another playlist by path, while the interface follows your selected language for dates.", "Smart playlists can reference another playlist by path"},
		{2, "Version 0.64.2 fixes scan failures and database lock contention on slow storage.", "Fix `database is locked` errors, UI freezes and failed scans on slow storage."},
		{2, "It also fixes scans on 32-bit builds with invalid track metadata.", "32-bit builds (armv5/6/7, 386)"},
		{2, "Security fixes sanitize download names and prevent an admin password from reaching logs.", "Stop writing the admin password to the log"},
	}
	r := narration{Cautions: []coverage{}}
	for _, item := range texts {
		if !strings.Contains(sources[item.index].Body, item.quote) {
			t.Fatalf("fixture quote absent: %s", item.quote)
		}
		r.Sentences = append(r.Sentences, sentence{Text: item.text, SourceID: sources[item.index].SourceID, Excerpt: item.quote})
	}
	for _, c := range cautions(sources) {
		index := 0
		for i, s := range r.Sentences {
			if s.SourceID == c.SourceID {
				index = i
				break
			}
		}
		r.Cautions = append(r.Cautions, coverage{CautionID: c.ID, SentenceIndex: &index})
	}
	return r
}

func response(status int, ct string, data []byte) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{ct}}, Body: io.NopCloser(bytes.NewReader(data))}
}

func testEngine(t *testing.T) (*engine, []releaseRecord, []source) {
	t.Helper()
	// Local fixtures must not inherit the hosting CI runner's Actions context.
	// Actions-specific tests establish their own context with actionsEnvironment.
	t.Setenv("GITHUB_ACTIONS", "false")
	t.Setenv("GH_TOKEN", "")
	t.Setenv("OPENAI_API_KEY", "")
	e, err := newEngine(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	e.openAIKey = "offline-test-key"
	records, sources := fixtures(t)
	e.command = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name == "git" {
			return []byte("test-commit"), nil
		}
		if len(args) == 1 && args[0] == "-version" {
			return nil, nil
		}
		if name == "ffprobe" {
			return []byte(`{"format":{"duration":"120.5"},"streams":[{"codec_name":"mp3"}]}`), nil
		}
		if name == "ffmpeg" {
			return nil, nil
		}
		return nil, errors.New("unexpected command")
	}
	e.client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "api.github.com" {
			if req.URL.Path == "/repos/"+repository+"/releases" {
				data, _ := json.Marshal(records)
				return response(200, "application/json", data), nil
			}
			if strings.Contains(req.URL.Path, "/actions/artifacts") {
				return response(200, "application/json", []byte(`{"artifacts":[]}`)), nil
			}
			for _, record := range records {
				if req.URL.Path == "/repos/"+repository+"/releases/tags/"+record.Tag || req.URL.Path == "/repos/"+repository+"/releases/"+strconv.FormatInt(record.ID, 10) {
					data, _ := json.Marshal(record)
					return response(200, "application/json", data), nil
				}
			}
		}
		if req.URL.Host == "api.openai.com" {
			if req.URL.Path == "/v1/responses" {
				data, _ := json.Marshal(exampleNarration(t, sources))
				body, _ := json.Marshal(map[string]any{"status": "completed", "usage": map[string]int{"input_tokens": 1000}, "output": []any{map[string]any{"type": "message", "content": []any{map[string]string{"type": "output_text", "text": string(data)}}}}})
				return response(200, "application/json", body), nil
			}
			if req.URL.Path == "/v1/audio/speech" {
				return response(200, "audio/mpeg", []byte("mock-mp3-data")), nil
			}
		}
		t.Fatalf("unmocked network request: %s", req.URL.Host+req.URL.Path)
		return nil, errors.New("unmocked request")
	})
	return e, records, sources
}

func audioOptions() options {
	return options{tags: "v0.64.0,v0.64.1,v0.64.2", mode: "audio", textModel: "gpt-6-luna", ttsModel: "gpt-4o-mini-tts-2025-12-15", voice: "onyx", allowPaid: true}
}

func TestTagSelection(t *testing.T) {
	for _, raw := range []string{"", "v0.64.0,", "v0.64.0,0.64.0", "v1.0.0,v2.0.0,v3.0.0,v4.0.0", "$(touch secret)", "../../foo", "v1.0.0\nmalicious", "v01.2.3", "v1.2.3١", "v99999999999.0.0"} {
		t.Run(raw, func(t *testing.T) {
			if _, err := parseTags(raw); err == nil {
				t.Fatal("unsafe tags accepted")
			}
		})
	}
	tags, err := parseTags("0.64.2, v0.64.0,v0.64.1")
	if err != nil || strings.Join(tags, ",") != "v0.64.0,v0.64.1,v0.64.2" {
		t.Fatal(tags, err)
	}
}

func TestReleaseEligibility(t *testing.T) {
	records, _ := fixtures(t)
	cases := map[string]func(*releaseRecord){"draft": func(r *releaseRecord) { r.Draft = true }, "prerelease": func(r *releaseRecord) { r.Prerelease = true }, "empty": func(r *releaseRecord) { r.Body = " " }, "unpublished": func(r *releaseRecord) { r.PublishedAt = "" }, "id": func(r *releaseRecord) { r.ID = 0 }, "oversized": func(r *releaseRecord) { r.Body = strings.Repeat("x", maxSourceBytes+1) }, "multiple tags": func(r *releaseRecord) { r.Tag = "v0.64.0,v0.64.1" }}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			r := records[0]
			mutate(&r)
			if _, err := normalizeRelease(r, false); err == nil {
				t.Fatal("ineligible release accepted")
			}
		})
	}
	r := records[0]
	r.Prerelease = true
	r.Tag = "v0.64.0-rc.1"
	if _, err := normalizeRelease(r, true); err != nil {
		t.Fatal(err)
	}
}

func TestInclusiveRangeSelection(t *testing.T) {
	e, _, _ := testEngine(t)
	sources, err := e.resolveLocal(t.Context(), options{from: "0.64.0", to: "v0.64.2"})
	if err != nil || len(sources) != 3 || sources[0].Tag != "v0.64.0" || sources[2].Tag != "v0.64.2" {
		t.Fatal(sources, err)
	}
	for _, o := range []options{{from: "v0.64.2", to: "v0.64.0"}, {from: "v0.63.0", to: "v0.64.2"}, {from: "v0.64.0"}, {tags: "v0.64.0", from: "v0.64.0", to: "v0.64.2"}, {from: "v0.64.0-rc.1", to: "v0.64.2"}} {
		if _, err := e.resolveLocal(t.Context(), o); err == nil {
			t.Fatal("invalid range accepted", o)
		}
	}
}

func TestRangeDiscardsDraftsAndFailsClosedAtLimit(t *testing.T) {
	e, records, _ := testEngine(t)
	draft := records[0]
	draft.ID = 999
	draft.Draft = true
	draft.Tag = "v0.64.0-rc.1"
	draft.Body = "private advisory never sent"
	data, _ := json.Marshal(append(records, draft))
	e.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) { return response(200, "application/json", data), nil })
	sources, err := e.resolveLocal(t.Context(), options{from: "v0.64.0", to: "v0.64.2", includePrereleases: true})
	if err != nil || len(sources) != 3 {
		t.Fatal(sources, err)
	}
	page := make([]releaseRecord, 100)
	for i := range page {
		page[i] = releaseRecord{Draft: true}
	}
	data, _ = json.Marshal(page)
	calls := 0
	e.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return response(200, "application/json", data), nil
	})
	if _, err := e.resolveLocal(t.Context(), options{from: "v0.64.0", to: "v0.64.2"}); err == nil || calls != 10 {
		t.Fatal("unbounded/incomplete range accepted", calls, err)
	}
}

func TestConfigurationAndCostLimits(t *testing.T) {
	e, _, sources := testEngine(t)
	cfg, err := reviewedConfig("gpt-6-luna", "gpt-4o-mini-tts-2025-12-15", "cedar", "audio")
	if err != nil {
		t.Fatal(err)
	}
	payload := e.textPayload(sources, cfg)
	cost, err := modeledCost(payload, cfg, "audio")
	if err != nil || cost == nil || *cost > maxCostUSD {
		t.Fatal(cost, err)
	}
	if payload["store"] != false || payload["tools"] != nil || payload["reasoning"] == nil {
		t.Fatal("unsafe text configuration")
	}
	if _, err := modeledCost(strings.Repeat("x", maxPromptBytes+1), cfg, "audio"); err == nil {
		t.Fatal("oversized prompt accepted")
	}
	hd, err := reviewedConfig("gpt-4.1-mini-2025-04-14", "tts-1-hd", "onyx", "audio")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := modeledCost(strings.Repeat("x", 60000), hd, "audio"); err == nil {
		t.Fatal("modeled dollar ceiling ignored")
	}
	for _, values := range [][3]string{{"", "", ""}, {"unreviewed", "tts-1", "onyx"}, {"gpt-6-luna", "unreviewed", "onyx"}, {"gpt-6-luna", "tts-1", "cedar"}, {"gpt-6-luna", "tts-1", "echo,fable"}} {
		if _, err := reviewedConfig(values[0], values[1], values[2], "audio"); err == nil {
			t.Fatal("invalid configuration accepted", values)
		}
	}
	if _, err := reviewedConfig("", "", "", "validate"); err != nil {
		t.Fatal(err)
	}
	legacy, err := reviewedConfig("gpt-6-luna", "tts-1", "onyx", "audio")
	if err != nil || legacy.Instructions != "" {
		t.Fatal(legacy, err)
	}
}

func TestCLIDryRunAndPaidGuards(t *testing.T) {
	t.Setenv("AUDIO_TEXT_MODEL", "gpt-6-luna")
	o, err := parseOptions([]string{"--tags", "0.64.2", "--text-model", "gpt-4.1-mini-2025-04-14", "--mode", "audio", "--dry-run", "--output", "out"}, io.Discard)
	if err != nil || o.mode != "validate" || o.textModel != "gpt-4.1-mini-2025-04-14" || o.output != "out" {
		t.Fatal(o, err)
	}
	if _, err := parseOptions([]string{"--openai-key", "never-pass-a-key"}, io.Discard); err == nil {
		t.Fatal("key argument accepted")
	}
	e, _, _ := testEngine(t)
	paid := audioOptions()
	paid.allowPaid = false
	if err := e.runLocal(t.Context(), paid, io.Discard); err == nil {
		t.Fatal("paid mode needs explicit flag")
	}
	paid.allowPaid = true
	e.openAIKey = ""
	if err := e.runLocal(t.Context(), paid, io.Discard); err == nil {
		t.Fatal("paid mode needs local environment key")
	}
}

func TestLocalValidationNeverContactsOpenAI(t *testing.T) {
	e, _, _ := testEngine(t)
	original := e.client.Transport
	e.client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "api.openai.com" {
			t.Fatal("validation contacted OpenAI")
		}
		return original.RoundTrip(req)
	})
	if err := e.runLocal(t.Context(), options{from: "v0.64.0", to: "v0.64.2", mode: "validate"}, io.Discard); err != nil {
		t.Fatal(err)
	}
	var m manifest
	if err := e.readJSON("manifest.json", &m); err != nil || m.TextRequests != 0 || m.SpeechRequests != 0 || m.Origin != "local" {
		t.Fatal(m, err)
	}
}

func TestLocalPaidPipelineAndDuplicateProtection(t *testing.T) {
	e, _, _ := testEngine(t)
	if err := e.runLocal(t.Context(), audioOptions(), io.Discard); err != nil {
		t.Fatal(err)
	}
	var m manifest
	if err := e.readJSON("manifest.json", &m); err != nil {
		t.Fatal(err)
	}
	if m.Status != "audio_validated" || m.TextRequests != 1 || m.SpeechRequests != 1 || m.AudioHash == "" || m.ScriptHash == "" || m.Duration != 120.5 {
		t.Fatal(m)
	}
	for _, name := range []string{"release-podcast.mp3", "transcript.txt", "evidence.json", "sources.json", "manifest.json"} {
		if _, err := os.Stat(filepath.Join(e.output, name)); err != nil {
			t.Fatal(err)
		}
	}
	if err := e.runLocal(t.Context(), audioOptions(), io.Discard); err == nil {
		t.Fatal("duplicate attempt permitted")
	}
	o := audioOptions()
	o.force = true
	if err := e.runLocal(t.Context(), o, io.Discard); err != nil {
		t.Fatal(err)
	}
}

func TestLocalLedgerPersistsFailuresAndLocksConcurrentAttempts(t *testing.T) {
	e, _, sources := testEngine(t)
	cfg, _ := reviewedConfig("gpt-6-luna", "tts-1", "onyx", "audio")
	m, err := e.prepare(t.Context(), sources, cfg, "audio", "local", false, false)
	if err != nil {
		t.Fatal(err)
	}
	release, err := e.reserveLocal(m, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.reserveLocal(m, true); err == nil {
		t.Fatal("force bypassed active lock")
	}
	release()
	if _, err := e.reserveLocal(m, false); err == nil {
		t.Fatal("reserved failed attempt retried")
	}
	release, err = e.reserveLocal(m, true)
	if err != nil {
		t.Fatal(err)
	}
	release()
}

func TestEvidenceAndConsequentialOmissions(t *testing.T) {
	_, sources := fixtures(t)
	if _, err := validateNarration(exampleNarration(t, sources), sources); err != nil {
		t.Fatal(err)
	}
	for _, term := range []string{"Back up", "resync", "experimental", "host HTTP", "opt-in", "host networking", "32-bit", "slow storage"} {
		t.Run(term, func(t *testing.T) {
			r := exampleNarration(t, sources)
			for i := range r.Sentences {
				r.Sentences[i].Text = strings.ReplaceAll(r.Sentences[i].Text, term, "some detail")
			}
			if _, err := validateNarration(r, sources); err == nil {
				t.Fatal("consequential qualifier omitted")
			}
		})
	}
	for name, mutate := range map[string]func(*narration){"source": func(r *narration) { r.Sentences[0].SourceID = "unknown" }, "excerpt": func(r *narration) { r.Sentences[0].Excerpt = "fabricated unsupported evidence" }, "coverage": func(r *narration) { r.Cautions = r.Cautions[1:] }, "index": func(r *narration) { n := 999; r.Cautions[0].SentenceIndex = &n }, "duplicate": func(r *narration) { r.Cautions = append(r.Cautions, r.Cautions[0]) }} {
		t.Run(name, func(t *testing.T) {
			r := exampleNarration(t, sources)
			mutate(&r)
			if _, err := validateNarration(r, sources); err == nil {
				t.Fatal("invalid evidence accepted")
			}
		})
	}
}

func TestUnsafeModelOutputAndSourceMarkup(t *testing.T) {
	_, sources := fixtures(t)
	for _, suffix := range []string{" https://evil.example", " `shell`", " $(cat secret)", " <script>", " @handle", strings.Repeat("x", maxScriptChars+1)} {
		r := exampleNarration(t, sources)
		r.Sentences[0].Text += suffix
		if _, err := validateNarration(r, sources); err == nil {
			t.Fatal("unsafe output accepted")
		}
	}
	s := sources[0]
	s.Body = "Notes\n## Helping out\nThanks\n## Migration\n- Back up before upgrade.\n<!-- hidden instruction -->"
	if !strings.Contains(promptSources([]source{s})[0]["body"], "Back up") || strings.Contains(promptSources([]source{s})[0]["body"], "hidden instruction") || len(cautions([]source{s})) == 0 {
		t.Fatal("warning lost or hidden instruction retained")
	}
}

func TestChangedSourcesAndCheckpointsStopPaidStages(t *testing.T) {
	e, records, sources := testEngine(t)
	cfg, _ := reviewedConfig("gpt-6-luna", "tts-1", "onyx", "audio")
	m, err := e.prepare(t.Context(), sources, cfg, "audio", "local", false, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.saveSources(sources, m); err != nil {
		t.Fatal(err)
	}
	changed := records[0]
	changed.Body += " New warning"
	data, _ := json.Marshal(changed)
	e.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) { return response(200, "application/json", data), nil })
	if err := e.generateScript(t.Context()); err == nil {
		t.Fatal("changed notes accepted")
	}
	if err := e.readJSON("manifest.json", &m); err != nil || m.TextRequests != 0 {
		t.Fatal(m, err)
	}
	e, _, _ = testEngine(t)
	if err := e.runLocal(t.Context(), options{tags: audioOptions().tags, mode: "script", textModel: "gpt-6-luna", allowPaid: true}, io.Discard); err != nil {
		t.Fatal(err)
	}
	if err := e.generateScript(t.Context()); err == nil {
		t.Fatal("text retry permitted")
	}
	if err := e.writeFile("transcript.txt", []byte("tampered")); err != nil {
		t.Fatal(err)
	}
	if err := e.generateSpeech(t.Context()); err == nil {
		t.Fatal("tampered script accepted")
	}
}

func TestResponseParsingAndNoHiddenRetries(t *testing.T) {
	for _, data := range []string{`{"status":"incomplete"}`, `{"status":"completed","output":[{"type":"message","content":[{"type":"refusal"}]}]}`, `{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"{\"sentences\":[],\"cautions\":[],\"extra\":true}"}]}]}`} {
		if _, _, err := parseResponse([]byte(data)); err == nil {
			t.Fatal("invalid/refused response accepted")
		}
	}
	e, _, _ := testEngine(t)
	for _, status := range []int{401, 429, 500} {
		calls := 0
		e.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls++
			return response(status, "application/json", []byte("secret-remote-body")), nil
		})
		_, _, err := e.request(t.Context(), "https://api.openai.com/v1/responses", "secret-key", map[string]string{"input": "public notes"}, 100)
		if err == nil || calls != 1 || strings.Contains(err.Error(), "secret") {
			t.Fatal(calls, err)
		}
	}
	calls := 0
	e.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("secret error details") })
	if _, _, err := e.request(t.Context(), "https://api.openai.com/v1/responses", "key", struct{}{}, 100); err == nil || calls != 1 || strings.Contains(err.Error(), "secret") {
		t.Fatal(calls, err)
	}
	e.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return response(200, "audio/mpeg", bytes.Repeat([]byte("x"), 101)), nil
	})
	if _, _, err := e.request(t.Context(), "https://api.openai.com/v1/audio/speech", "key", struct{}{}, 100); err == nil {
		t.Fatal("response limit ignored")
	}
}

func TestCredentialsNeverFollowRedirects(t *testing.T) {
	called := false
	e, err := newEngine(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	e.client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "evil.example" {
			called = true
			return response(200, "application/json", nil), nil
		}
		r := response(http.StatusFound, "application/json", nil)
		r.Header.Set("Location", "https://evil.example/")
		return r, nil
	})
	if _, _, err := e.request(t.Context(), "https://api.openai.com/v1/responses", "never-forward-me", nil, 100); err == nil || called {
		t.Fatal("authenticated redirect followed", err)
	}
}

func actionsEnvironment(t *testing.T, record releaseRecord, manual bool) {
	t.Helper()
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("GITHUB_REPOSITORY", repository)
	t.Setenv("GITHUB_RUN_ID", "123")
	t.Setenv("GITHUB_RUN_ATTEMPT", "1")
	t.Setenv("AUDIO_ENABLED", "true")
	t.Setenv("AUDIO_KEY_CONFIGURED", "true")
	t.Setenv("AUDIO_TEXT_MODEL", "gpt-6-luna")
	t.Setenv("AUDIO_TTS_MODEL", "tts-1")
	t.Setenv("AUDIO_VOICE", "onyx")
	dir := t.TempDir()
	ev := map[string]any{"repository": map[string]string{"default_branch": "master"}, "action": "published", "release": record}
	if manual {
		t.Setenv("GITHUB_EVENT_NAME", "workflow_dispatch")
		t.Setenv("GITHUB_REF", "refs/heads/master")
		ev["inputs"] = map[string]string{"tags": "v0.64.0,v0.64.1,v0.64.2", "mode": "audio"}
	} else {
		t.Setenv("GITHUB_EVENT_NAME", "release")
		t.Setenv("GITHUB_REF", "refs/tags/"+record.Tag)
	}
	data, _ := json.Marshal(ev)
	eventFile := filepath.Join(dir, "event.json")
	if err := os.WriteFile(eventFile, data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_EVENT_PATH", eventFile)
	output := filepath.Join(dir, "outputs")
	if err := os.WriteFile(output, nil, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_OUTPUT", output)
}

func TestActionsEventIdentityAndTrustGuards(t *testing.T) {
	e, records, _ := testEngine(t)
	actionsEnvironment(t, records[2], false)
	if err := e.runGitHub(t.Context(), "prepare"); err != nil {
		t.Fatal(err)
	}
	var sources []source
	if err := e.readJSON("sources.json", &sources); err != nil || len(sources) != 1 || sources[0].ID != records[2].ID {
		t.Fatal(sources, err)
	}
	for name, value := range map[string]string{"GITHUB_REPOSITORY": "attacker/navidrome", "GITHUB_EVENT_NAME": "pull_request", "GITHUB_ACTIONS": "false"} {
		previous := os.Getenv(name)
		t.Setenv(name, value)
		if _, err := githubEvent(); err == nil {
			t.Fatal("untrusted context accepted", name)
		}
		t.Setenv(name, previous)
	}
	actionsEnvironment(t, records[0], true)
	t.Setenv("GITHUB_REF", "refs/heads/untrusted")
	if err := e.runGitHub(t.Context(), "prepare"); err == nil {
		t.Fatal("non-default branch accepted")
	}
	if err := run(t.Context(), []string{"--tags", "v0.64.0", "--mode", "audio", "--allow-paid"}, io.Discard, io.Discard); err == nil {
		t.Fatal("Actions used local entrypoint")
	}
}

func TestActionsEnablementAndConfigChanges(t *testing.T) {
	e, records, _ := testEngine(t)
	actionsEnvironment(t, records[0], true)
	t.Setenv("AUDIO_KEY_CONFIGURED", "false")
	if err := e.runGitHub(t.Context(), "prepare"); err == nil {
		t.Fatal("missing secret accepted")
	}
	t.Setenv("AUDIO_KEY_CONFIGURED", "true")
	t.Setenv("AUDIO_ENABLED", "false")
	if err := e.runGitHub(t.Context(), "prepare"); err == nil {
		t.Fatal("disabled generation accepted")
	}
	t.Setenv("AUDIO_ENABLED", "true")
	if err := e.runGitHub(t.Context(), "prepare"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AUDIO_VOICE", "nova")
	if err := e.runGitHub(t.Context(), "script"); err == nil {
		t.Fatal("configuration changed between stages")
	}
	t.Setenv("AUDIO_VOICE", "onyx")
	t.Setenv("GITHUB_RUN_ATTEMPT", "2")
	if err := e.runGitHub(t.Context(), "script"); err == nil {
		t.Fatal("checkpoint from another attempt accepted")
	}
	t.Setenv("GITHUB_RUN_ATTEMPT", "1")
	var m manifest
	if err := e.readJSON("manifest.json", &m); err != nil {
		t.Fatal(err)
	}
	m.Origin = "local"
	if err := e.writeJSON("manifest.json", m); err != nil {
		t.Fatal(err)
	}
	if err := e.runGitHub(t.Context(), "script"); err == nil {
		t.Fatal("local checkpoint accepted by Actions")
	}
}

func TestArtifactDuplicatesExpiryAndLookupBounds(t *testing.T) {
	e, _, _ := testEngine(t)
	for _, expired := range []bool{false, true} {
		body, _ := json.Marshal(map[string]any{"artifacts": []any{map[string]any{"name": "key-123-1", "expired": expired}}})
		e.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) { return response(200, "application/json", body), nil })
		duplicate, err := e.duplicateArtifact(t.Context(), "key")
		if err != nil || duplicate == expired {
			t.Fatal(duplicate, err)
		}
	}
	artifacts := make([]map[string]any, 100)
	for i := range artifacts {
		artifacts[i] = map[string]any{"name": "other", "expired": false}
	}
	body, _ := json.Marshal(map[string]any{"artifacts": artifacts})
	calls := 0
	e.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return response(200, "application/json", body), nil
	})
	if _, err := e.duplicateArtifact(t.Context(), "key"); err == nil || calls != 100 {
		t.Fatal("ledger lookup not bounded", calls, err)
	}
}

func TestInvalidMP3NeverBecomesOutput(t *testing.T) {
	for _, metadata := range []string{`{"format":{"duration":"nan"},"streams":[{"codec_name":"mp3"}]}`, `{"format":{"duration":"0"},"streams":[{"codec_name":"mp3"}]}`, `{"format":{"duration":"120"},"streams":[{"codec_name":"aac"}]}`} {
		e, _, _ := testEngine(t)
		original := e.command
		e.command = func(ctx context.Context, name string, args ...string) ([]byte, error) {
			if name == "ffprobe" && len(args) > 1 {
				return []byte(metadata), nil
			}
			return original(ctx, name, args...)
		}
		if err := e.runLocal(t.Context(), audioOptions(), io.Discard); err == nil {
			t.Fatal("invalid MP3 accepted")
		}
		if _, err := os.Stat(filepath.Join(e.output, "release-podcast.mp3")); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("invalid output exists", err)
		}
		if _, err := os.Stat(filepath.Join(e.output, "audio.tmp")); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("temporary output retained", err)
		}
	}
}

func TestMissingMediaToolsStopBeforeOpenAI(t *testing.T) {
	e, _, _ := testEngine(t)
	e.command = func(context.Context, string, ...string) ([]byte, error) { return nil, errors.New("tool missing") }
	e.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("network should not start before media preflight")
		return nil, errors.New("unexpected")
	})
	if err := e.runLocal(t.Context(), audioOptions(), io.Discard); err == nil {
		t.Fatal("missing media tools accepted")
	}
}

func TestRealMP3DecodeWithMockedOpenAI(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg unavailable")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe unavailable")
	}
	e, _, _ := testEngine(t)
	file := filepath.Join(t.TempDir(), "tone.mp3")
	if _, err := exec.CommandContext(t.Context(), "ffmpeg", "-v", "error", "-f", "lavfi", "-i", "sine=frequency=440", "-t", "0.2", "-c:a", "libmp3lame", file).Output(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	original := e.client.Transport
	e.client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "api.openai.com" && req.URL.Path == "/v1/audio/speech" {
			return response(200, "audio/mpeg", data), nil
		}
		return original.RoundTrip(req)
	})
	e.command = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return exec.CommandContext(ctx, name, args...).Output()
	}
	if err := e.runLocal(t.Context(), audioOptions(), io.Discard); err != nil {
		t.Fatal(err)
	}
	var m manifest
	if err := e.readJSON("manifest.json", &m); err != nil || !m.DurationNeedsReview || m.Duration <= 0 {
		t.Fatal(m, err)
	}
}
