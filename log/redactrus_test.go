package log

import (
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var h = &Hook{}

type levelsTest struct {
	name           string
	acceptedLevels []logrus.Level
	expected       []logrus.Level
	description    string
}

func TestLevels(t *testing.T) {
	tests := []levelsTest{
		{
			name:           "undefinedAcceptedLevels",
			acceptedLevels: []logrus.Level{},
			expected:       logrus.AllLevels,
			description:    "All logrus levels expected, but did not receive them",
		},
		{
			name:           "definedAcceptedLevels",
			acceptedLevels: []logrus.Level{logrus.InfoLevel},
			expected:       []logrus.Level{logrus.InfoLevel},
			description:    "Logrus Info level expected, but did not receive that.",
		},
	}

	for _, test := range tests {
		fn := func(t *testing.T) {
			h.AcceptedLevels = test.acceptedLevels
			levels := h.Levels()
			assert.Equal(t, test.expected, levels, test.description)
		}

		t.Run(test.name, fn)
	}
}

type levelThresholdTest struct {
	name        string
	level       logrus.Level
	expected    []logrus.Level
	description string
}

// levelThreshold returns a []logrus.Level including all levels
// above and including the level given. If the provided level does not exit,
// an empty slice is returned.
func levelThreshold(l logrus.Level) []logrus.Level {
	//nolint
	if l < 0 || int(l) > len(logrus.AllLevels) {
		return []logrus.Level{}
	}
	return logrus.AllLevels[:l+1]
}

func TestLevelThreshold(t *testing.T) {
	tests := []levelThresholdTest{
		{
			name:        "unknownLogLevel",
			level:       logrus.Level(100),
			expected:    []logrus.Level{},
			description: "An empty Level slice was expected but was not returned",
		},
		{
			name:        "errorLogLevel",
			level:       logrus.ErrorLevel,
			expected:    []logrus.Level{logrus.PanicLevel, logrus.FatalLevel, logrus.ErrorLevel},
			description: "The panic, fatal, and error levels were expected but were not returned",
		},
	}

	for _, test := range tests {
		fn := func(t *testing.T) {
			levels := levelThreshold(test.level)
			assert.Equal(t, test.expected, levels, test.description)
		}

		t.Run(test.name, fn)
	}
}

func TestInvalidRegex(t *testing.T) {
	e := &logrus.Entry{}
	h = &Hook{RedactionList: []string{"\\"}}
	err := h.Fire(e)

	assert.NotNil(t, err)
}

type EntryDataValuesTest struct {
	name          string
	redactionList []string
	logFields     logrus.Fields
	expected      logrus.Fields
	description   string //nolint
}

// Test that any occurrence of a redaction pattern
// in the values of the entry's data fields is redacted.
func TestEntryDataValues(t *testing.T) {
	tests := []EntryDataValuesTest{
		{
			name:          "match on key",
			redactionList: []string{"Password"},
			logFields:     logrus.Fields{"Password": "password123!"},
			expected:      logrus.Fields{"Password": "[REDACTED]"},
			description:   "Password value should have been redacted, but was not.",
		},
		{
			name:          "string value",
			redactionList: []string{"William"},
			logFields:     logrus.Fields{"Description": "His name is William"},
			expected:      logrus.Fields{"Description": "His name is [REDACTED]"},
			description:   "William should have been redacted, but was not.",
		},
		{
			name:          "map value",
			redactionList: []string{"William"},
			logFields:     logrus.Fields{"Description": map[string]string{"name": "His name is William"}},
			expected:      logrus.Fields{"Description": "map[name:His name is [REDACTED]]"},
			description:   "William should have been redacted, but was not.",
		},
	}

	for _, test := range tests {
		fn := func(t *testing.T) {
			logEntry := &logrus.Entry{
				Data: test.logFields,
			}
			h = &Hook{RedactionList: test.redactionList}
			err := h.Fire(logEntry)

			assert.Nil(t, err)
			assert.Equal(t, test.expected, logEntry.Data)
		}
		t.Run(test.name, fn)
	}
}

// Test that any occurrence of a redaction pattern
// in the entry's Message field is redacted.
func TestEntryMessage(t *testing.T) {
	logEntry := &logrus.Entry{
		Message: "Secret Password: password123!",
	}
	h = &Hook{RedactionList: []string{`(Password: ).*`}}
	err := h.Fire(logEntry)

	assert.Nil(t, err)
	assert.Equal(t, "Secret Password: [REDACTED]", logEntry.Message)
}

type namedString string

func TestFireRedactsNamedStringTypes(t *testing.T) {
	hook := &Hook{RedactionList: []string{"(secret=)[^&]+"}}
	e := &logrus.Entry{Data: logrus.Fields{"code": namedString("not_found"), "url": namedString("/x?secret=abc")}}

	assert.NotPanics(t, func() { _ = hook.Fire(e) })
	assert.Equal(t, "not_found", e.Data["code"])
	assert.Equal(t, "/x?secret=[REDACTED]", e.Data["url"])
}

func TestFireRedactsContextSecrets(t *testing.T) {
	ctx := WithSecrets(t.Context(), "s3cr3t")
	ctx = WithSecrets(ctx, "", "other-secret")
	e := &logrus.Entry{
		Context: ctx,
		Message: "value s3cr3t in message",
		Data: logrus.Fields{
			"str":   "has s3cr3t",
			"named": namedString("named other-secret"),
			"args":  map[string]any{"p0": "s3cr3t", "p1": "plain"},
			"error": errors.New("failed with other-secret"),
			"num":   42,
			"clean": namedString("untouched"),
		},
	}

	assert.Nil(t, (&Hook{}).Fire(e))
	assert.Equal(t, "value [REDACTED] in message", e.Message)
	assert.Equal(t, "has [REDACTED]", e.Data["str"])
	assert.Equal(t, "named [REDACTED]", e.Data["named"])
	assert.Equal(t, "map[p0:[REDACTED] p1:plain]", e.Data["args"])
	assert.Equal(t, "failed with [REDACTED]", e.Data["error"])
	assert.Equal(t, 42, e.Data["num"])
	assert.Equal(t, namedString("untouched"), e.Data["clean"])
}

func TestFireWithoutContextSecretsLeavesEntryUnchanged(t *testing.T) {
	args := map[string]any{"p0": "value"}
	e := &logrus.Entry{Context: t.Context(), Message: "value", Data: logrus.Fields{"str": "value", "args": args}}

	assert.Nil(t, (&Hook{}).Fire(e))
	assert.Equal(t, "value", e.Message)
	assert.Equal(t, logrus.Fields{"str": "value", "args": args}, e.Data)
}

func TestFireRedactsLongerSecretsFirst(t *testing.T) {
	e := &logrus.Entry{Context: WithSecrets(t.Context(), "abc", "abcdef"), Message: "abcdef"}

	assert.Nil(t, (&Hook{}).Fire(e))
	assert.Equal(t, "[REDACTED]", e.Message)
}
