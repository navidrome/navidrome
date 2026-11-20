package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

type sentence struct {
	Text     string `json:"text"`
	SourceID string `json:"source_id"`
	Excerpt  string `json:"excerpt"`
}
type coverage struct {
	CautionID     string `json:"caution_id"`
	SentenceIndex *int   `json:"sentence_index"`
}
type narration struct {
	Sentences []sentence `json:"sentences"`
	Cautions  []coverage `json:"cautions"`
}
type caution struct {
	ID       string `json:"caution_id"`
	SourceID string `json:"source_id"`
	Excerpt  string `json:"excerpt"`
}

var htmlComments = regexp.MustCompile(`(?s)<!--.*?-->`)
var migrationHeader = regexp.MustCompile(`(?i)breaking|migration`)
var securityWord = regexp.MustCompile(`(?i)security`)
var qualifierWord = regexp.MustCompile(`(?i)back.?up|re-?sync|experimental|opt-in|disabled|host networking`)
var unsafeNarration = regexp.MustCompile("(?i)https?://|www\\.|[@`<>{}\\[\\]#]|\\$\\(|[\\x00-\\x08\\x0b-\\x1f]")

func visibleNotes(body string) string { return htmlComments.ReplaceAllString(body, "") }

func promptSources(sources []source) []map[string]string {
	result := make([]map[string]string, 0, len(sources))
	for _, s := range sources {
		result = append(result, map[string]string{"source_id": s.SourceID, "tag": s.Tag, "body": visibleNotes(s.Body)})
	}
	return result
}

func cautions(sources []source) []caution {
	result := []caution{}
	for _, s := range sources {
		migration, securityAdded := false, false
		for _, line := range strings.Split(visibleNotes(s.Body), "\n") {
			if strings.HasPrefix(line, "## ") {
				migration = migrationHeader.MatchString(line)
			}
			mandatory := migration && strings.HasPrefix(line, "- ")
			if securityWord.MatchString(line) && !securityAdded {
				mandatory = true
				securityAdded = true
			}
			if mandatory || qualifierWord.MatchString(line) {
				result = append(result, caution{ID: fmt.Sprintf("c%d", len(result)), SourceID: s.SourceID, Excerpt: line})
			}
		}
	}
	return result
}

func narrationSchema() map[string]any {
	stringType := map[string]string{"type": "string"}
	return map[string]any{"type": "object", "additionalProperties": false, "required": []string{"sentences", "cautions"}, "properties": map[string]any{
		"sentences": map[string]any{"type": "array", "items": map[string]any{"type": "object", "additionalProperties": false, "required": []string{"text", "source_id", "excerpt"}, "properties": map[string]any{"text": stringType, "source_id": stringType, "excerpt": stringType}}},
		"cautions":  map[string]any{"type": "array", "items": map[string]any{"type": "object", "additionalProperties": false, "required": []string{"caution_id", "sentence_index"}, "properties": map[string]any{"caution_id": stringType, "sentence_index": map[string]string{"type": "integer"}}}},
	}}
}

func parseResponse(data []byte) (narration, json.RawMessage, error) {
	var response struct {
		Status string          `json:"status"`
		Usage  json.RawMessage `json:"usage"`
		Output []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	var result narration
	if json.Unmarshal(data, &response) != nil || response.Status != "completed" {
		return result, nil, errors.New("text response invalid, incomplete, or refused")
	}
	var texts []string
	for _, output := range response.Output {
		if output.Type != "message" {
			continue
		}
		for _, part := range output.Content {
			if part.Type == "refusal" {
				return result, nil, errors.New("text response refused")
			}
			if part.Type == "output_text" {
				texts = append(texts, part.Text)
			}
		}
	}
	if len(texts) != 1 {
		return result, nil, errors.New("expected one structured narration")
	}
	if err := strictDecode([]byte(texts[0]), &result); err != nil {
		return result, nil, err
	}
	return result, response.Usage, nil
}

func validateNarration(result narration, sources []source) (string, error) {
	if result.Cautions == nil {
		return "", errors.New("narration requires an explicit caution coverage array")
	}
	if len(result.Sentences) < 1 || len(result.Sentences) > 35 {
		return "", errors.New("invalid narration sentence count")
	}
	byID := map[string]source{}
	for _, s := range sources {
		byID[s.SourceID] = s
	}
	texts := []string{}
	for _, s := range result.Sentences {
		original, ok := byID[s.SourceID]
		if !ok || strings.TrimSpace(s.Text) == "" || len(s.Excerpt) < 12 || !strings.Contains(visibleNotes(original.Body), s.Excerpt) {
			return "", errors.New("sentence evidence is absent from its published source")
		}
		texts = append(texts, strings.TrimSpace(s.Text))
	}
	required := map[string]caution{}
	for _, c := range cautions(sources) {
		required[c.ID] = c
	}
	covered := map[string]bool{}
	for _, c := range result.Cautions {
		r, ok := required[c.CautionID]
		if !ok || covered[c.CautionID] || c.SentenceIndex == nil || *c.SentenceIndex < 0 || *c.SentenceIndex >= len(result.Sentences) {
			return "", errors.New("invalid or duplicate caution reference")
		}
		if result.Sentences[*c.SentenceIndex].SourceID != r.SourceID {
			return "", errors.New("caution mapped to the wrong release")
		}
		covered[c.CautionID] = true
	}
	if len(covered) != len(required) {
		return "", errors.New("missing required migration/security/qualifier coverage")
	}
	text := intro + "\n\n" + strings.Join(texts, " ") + "\n\n" + closing
	if len(strings.Fields(text)) < 100 || len(strings.Fields(text)) > 280 || utf8.RuneCountInString(text) > maxScriptChars {
		return "", errors.New("narration must be 100-280 words and at most 2500 characters")
	}
	if unsafeNarration.MatchString(text) {
		return "", errors.New("narration contains markup, URL, handle, or executable content")
	}
	ascii := 0
	for _, r := range text {
		if r < 128 {
			ascii++
		}
	}
	if float64(ascii)/float64(utf8.RuneCountInString(text)) < 0.95 {
		return "", errors.New("narration must be English plain text")
	}
	for _, s := range sources {
		if !strings.Contains(text, strings.TrimPrefix(s.Tag, "v")) {
			return "", errors.New("narration must identify every selected release version")
		}
	}
	if err := validateQualifiers(result.Sentences, sources); err != nil {
		return "", err
	}
	return text + "\n", nil
}

type qualifierRule struct {
	trigger string
	terms   []string
}

var qualifierRules = []qualifierRule{
	{`back up your database before upgrading`, []string{`back.?up`, `database`, `before.{0,40}upgrad`}},
	{`may need to re-sync`, []string{`re.?sync`}},
	{`experimental Jellyfin`, []string{`experimental`, `enabl|opt.in|default.off|disabled by default`}},
	{`Plugin authors.*?Extism.*?disabled`, []string{`plugin`, `host.{0,20}HTTP|host.{0,20}network`, `private|loopback|LAN`}},
	{`security release.*?Upgrade`, []string{`security`, `upgrad`}},
	{`opt-in LAN auto-discovery`, []string{`opt.in`, `Docker`, `host networking`}},
	{`slow storage`, []string{`slow.{0,20}storage`, `scan|lock`}},
	{`32-bit builds`, []string{`32.bit`, `scan`}},
}

func validateQualifiers(sentences []sentence, sources []source) error {
	// These are lexical checks, not proof of semantic entailment. Human review
	// remains necessary even when all evidence and safety checks pass.
	for _, s := range sources {
		body := strings.NewReplacer("*", "", "`", "").Replace(visibleNotes(s.Body))
		for _, rule := range qualifierRules {
			if !regexp.MustCompile("(?is)" + rule.trigger).MatchString(body) {
				continue
			}
			matched := false
			for _, sentence := range sentences {
				if sentence.SourceID != s.SourceID {
					continue
				}
				all := true
				for _, term := range rule.terms {
					all = all && regexp.MustCompile("(?i)"+term).MatchString(sentence.Text)
				}
				matched = matched || all
			}
			if !matched {
				return errors.New("narration omits a consequential source qualifier or action")
			}
		}
	}
	return nil
}
