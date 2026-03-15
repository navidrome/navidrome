package log

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestJournalFormatterPrefixes(t *testing.T) {
	inner := &logrus.TextFormatter{
		DisableTimestamp: true,
		DisableColors:   true,
	}
	formatter := &journalFormatter{inner: inner}

	tests := []struct {
		level          logrus.Level
		expectedPrefix string
	}{
		{logrus.ErrorLevel, "<3>"},
		{logrus.WarnLevel, "<4>"},
		{logrus.InfoLevel, "<6>"},
		{logrus.DebugLevel, "<7>"},
		{logrus.TraceLevel, "<7>"},
		{logrus.FatalLevel, "<2>"},
		{logrus.PanicLevel, "<0>"},
		{logrus.Level(99), "<6>"}, // unknown level defaults to info
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			entry := &logrus.Entry{
				Logger:  logrus.New(),
				Level:   tt.level,
				Message: "test message",
				Data:    logrus.Fields{},
			}
			out, err := formatter.Format(entry)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := string(out)
			if len(got) < len(tt.expectedPrefix) || got[:len(tt.expectedPrefix)] != tt.expectedPrefix {
				t.Errorf("expected prefix %q, got %q", tt.expectedPrefix, got)
			}
		})
	}
}
