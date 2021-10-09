package model

import (
	"encoding/json"
	"errors"
)

type SmartPlaylist struct {
	RuleGroup
	Order string `json:"order,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

type RuleGroup struct {
	Combinator string `json:"combinator"`
	Rules      Rules  `json:"rules"`
}

type Rules []IRule

type IRule interface {
	Fields() []string
}

type Rule struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value,omitempty"`
}

func (r Rule) Fields() []string {
	return []string{r.Field}
}

func (rg RuleGroup) Fields() []string {
	var result []string
	unique := map[string]struct{}{}
	for _, r := range rg.Rules {
		for _, f := range r.Fields() {
			if _, added := unique[f]; !added {
				result = append(result, f)
				unique[f] = struct{}{}
			}
		}
	}
	return result
}

func (rs *Rules) UnmarshalJSON(data []byte) error {
	var rawRules []json.RawMessage
	if err := json.Unmarshal(data, &rawRules); err != nil {
		return err
	}
	rules := make(Rules, len(rawRules))
	for i, rawRule := range rawRules {
		var r Rule
		if err := json.Unmarshal(rawRule, &r); err == nil && r.Field != "" {
			rules[i] = r
			continue
		}
		var g RuleGroup
		if err := json.Unmarshal(rawRule, &g); err == nil && g.Combinator != "" {
			rules[i] = g
			continue
		}
		return errors.New("Invalid json. Neither a Rule nor a RuleGroup: " + string(rawRule))
	}
	*rs = rules
	return nil
}

var SmartPlaylistFields = []string{
	"title",
	"album",
	"artist",
	"albumartist",
	"albumartwork",
	"tracknumber",
	"discnumber",
	"year",
	"size",
	"compilation",
	"dateadded",
	"datemodified",
	"discsubtitle",
	"comment",
	"lyrics",
	"sorttitle",
	"sortalbum",
	"sortartist",
	"sortalbumartist",
	"albumtype",
	"albumcomment",
	"catalognumber",
	"filepath",
	"filetype",
	"duration",
	"bitrate",
	"bpm",
	"channels",
	"genre",
	"loved",
	"lastplayed",
	"playcount",
	"rating",
}
