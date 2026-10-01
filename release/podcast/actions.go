package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
)

type event struct {
	Action     string        `json:"action"`
	Release    releaseRecord `json:"release"`
	Repository struct {
		DefaultBranch string `json:"default_branch"`
	} `json:"repository"`
	Inputs struct {
		Tags               string `json:"tags"`
		Mode               string `json:"mode"`
		IncludePrereleases string `json:"include_prereleases"`
		Force              string `json:"force_regenerate"`
	} `json:"inputs"`
}

func githubEvent() (event, error) {
	var ev event
	if os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("GITHUB_REPOSITORY") != repository {
		return ev, errors.New("Actions entrypoint requires the navidrome repository context")
	}
	data, err := os.ReadFile(os.Getenv("GITHUB_EVENT_PATH")) // #nosec G703 -- event path is provided by the verified GitHub runner context, never release data.
	if err != nil || len(data) > 8<<20 || json.Unmarshal(data, &ev) != nil {
		return ev, errors.New("invalid Actions event")
	}
	switch os.Getenv("GITHUB_EVENT_NAME") {
	case "workflow_dispatch":
		if ev.Repository.DefaultBranch == "" || os.Getenv("GITHUB_REF") != "refs/heads/"+ev.Repository.DefaultBranch {
			return ev, errors.New("manual Actions generation must use the default branch")
		}
	case "release":
		if ev.Action != "published" {
			return ev, errors.New("expected a release published event")
		}
		if _, err := normalizeRelease(ev.Release, false); err != nil {
			return ev, err
		}
	default:
		return ev, errors.New("unsupported Actions event")
	}
	return ev, nil
}

func (e *engine) duplicateArtifact(ctx context.Context, reservation string) (bool, error) {
	for page := 1; page <= 100; page++ {
		var response struct {
			Artifacts []struct {
				Name    string `json:"name"`
				Expired bool   `json:"expired"`
			} `json:"artifacts"`
		}
		if err := e.github(ctx, fmt.Sprintf("/actions/artifacts?name=%s&per_page=100&page=%d", url.QueryEscape(reservation), page), &response); err != nil {
			return false, err
		}
		for _, a := range response.Artifacts {
			if !a.Expired && a.Name == reservation {
				return true, nil
			}
		}
		if len(response.Artifacts) < 100 {
			return false, nil
		}
	}
	return false, errors.New("matching reservation ledger exceeds lookup limit; manual review required")
}

func actionsOutput(key, value string) error {
	f, err := os.OpenFile(os.Getenv("GITHUB_OUTPUT"), os.O_APPEND|os.O_WRONLY, 0600) // #nosec G703 -- append only to the runner-provided Actions output file.
	if err != nil {
		return errors.New("Actions output file unavailable")
	}
	defer f.Close()
	if _, err := fmt.Fprintf(f, "%s=%s\n", key, value); err != nil {
		return errors.New("cannot write Actions output")
	}
	return nil
}

func (e *engine) runGitHub(ctx context.Context, stage string) error {
	ev, err := githubEvent()
	if err != nil {
		return err
	}
	switch stage {
	case "script":
		return e.generateScript(ctx)
	case "speech":
		return e.generateSpeech(ctx)
	case "prepare":
		return e.prepareGitHub(ctx, ev)
	default:
		return errors.New("invalid Actions stage")
	}
}

func (e *engine) prepareGitHub(ctx context.Context, ev event) error {
	manual := os.Getenv("GITHUB_EVENT_NAME") == "workflow_dispatch"
	mode := "audio"
	allow, force := false, false
	if manual {
		mode = ev.Inputs.Mode
		if mode == "" {
			mode = "validate"
		}
		allow = ev.Inputs.IncludePrereleases == "true"
		force = ev.Inputs.Force == "true"
	}
	if mode != "validate" && mode != "script" && mode != "audio" {
		return errors.New("invalid manual mode")
	}
	if mode != "validate" && (os.Getenv("AUDIO_ENABLED") != "true" || os.Getenv("AUDIO_KEY_CONFIGURED") != "true") {
		return errors.New("paid Actions modes require explicit enablement and OPENAI_API_KEY in Secrets")
	}
	c, err := reviewedConfig(os.Getenv("AUDIO_TEXT_MODEL"), os.Getenv("AUDIO_TTS_MODEL"), os.Getenv("AUDIO_VOICE"), mode)
	if err != nil {
		return err
	}
	if mode == "audio" {
		if err := e.mediaTools(ctx); err != nil {
			return err
		}
	}
	var sources []source
	if manual {
		sources, err = e.resolveLocal(ctx, options{tags: ev.Inputs.Tags, includePrereleases: allow})
	} else {
		var record releaseRecord
		err = e.github(ctx, fmt.Sprintf("/releases/%d", ev.Release.ID), &record)
		if err == nil {
			var s source
			s, err = normalizeRelease(record, false)
			if err == nil && (s.ID != ev.Release.ID || s.Tag != ev.Release.Tag) {
				err = errors.New("release identity changed")
			}
			sources = []source{s}
		}
	}
	if err != nil {
		return err
	}
	m, err := e.prepare(ctx, sources, c, mode, "github", allow, force)
	if err != nil {
		return err
	}
	m.RunID = os.Getenv("GITHUB_RUN_ID")
	m.RunAttempt = os.Getenv("GITHUB_RUN_ATTEMPT")
	m.EventSHA = os.Getenv("GITHUB_SHA")
	if mode != "validate" {
		repeated, err := e.duplicateArtifact(ctx, m.Reservation)
		if err != nil {
			return err
		}
		if repeated && !force {
			m.Status = "duplicate"
		}
	}
	if err := e.saveSources(sources, m); err != nil {
		return err
	}
	if err := actionsOutput("reservation", m.Reservation); err != nil {
		return err
	}
	if err := actionsOutput("prepared", "true"); err != nil {
		return err
	}
	return actionsOutput("generate", fmt.Sprint(mode != "validate" && m.Status != "duplicate"))
}
