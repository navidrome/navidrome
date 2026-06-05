package log

// Copied from https://github.com/whuang8/redactrus (MIT License)
// Copyright (c) 2018 William Huang

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/sirupsen/logrus"
)

// Hook is a logrus hook for redacting information from logs
type Hook struct {
	// Messages with a log level not contained in this array
	// will not be dispatched. If empty, all messages will be dispatched.
	AcceptedLevels []logrus.Level
	RedactionList  []string
	redactionKeys  []*regexp.Regexp
}

// Levels returns the user defined AcceptedLevels
// If AcceptedLevels is empty, all logrus levels are returned
func (h *Hook) Levels() []logrus.Level {
	if len(h.AcceptedLevels) == 0 {
		return logrus.AllLevels
	}
	return h.AcceptedLevels
}

// Fire redacts values in a log Entry that match
// with keys defined in the RedactionList
func (h *Hook) Fire(e *logrus.Entry) error {
	if err := h.initRedaction(); err != nil {
		return err
	}
	for _, re := range h.redactionKeys {
		// Redact based on key matching in Data fields
		for k, v := range e.Data {
			if re.MatchString(k) {
				e.Data[k] = "[REDACTED]"
				continue
			}
			if v == nil {
				continue
			}
			switch reflect.TypeOf(v).Kind() {
			case reflect.String:
				e.Data[k] = re.ReplaceAllString(v.(string), "$1[REDACTED]$2")
				continue
			case reflect.Map:
				s := fmt.Sprintf("%+v", v)
				e.Data[k] = re.ReplaceAllString(s, "$1[REDACTED]$2")
				continue
			}
		}

		// Redact based on text matching in the Message field
		e.Message = re.ReplaceAllString(e.Message, "$1[REDACTED]$2")
	}

	return nil
}

func (h *Hook) initRedaction() error {
	if len(h.redactionKeys) == 0 {
		for _, redactionKey := range h.RedactionList {
			re, err := regexp.Compile(redactionKey)
			if err != nil {
				return err
			}
			h.redactionKeys = append(h.redactionKeys, re)
		}
	}
	return nil
}

func (h *Hook) redact(msg string) (string, error) {
	if err := h.initRedaction(); err != nil {
		return msg, err
	}
	for _, re := range h.redactionKeys {
		msg = re.ReplaceAllString(msg, "$1[REDACTED]$2")
	}

	return msg, nil
}
