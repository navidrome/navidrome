package scheduler

import (
	"github.com/navidrome/navidrome/log"
)

type logger struct{}

func (l *logger) Info(msg string, keysAndValues ...interface{}) {
	args := []interface{}{
		"Scheduler: " + msg,
	}
	args = append(args, keysAndValues...)
	log.Debug(args...)
}

func (l *logger) Error(err error, msg string, keysAndValues ...interface{}) {
	args := []interface{}{
		"Scheduler: " + msg,
	}
	args = append(args, keysAndValues...)
	args = append(args, err)
	log.Error(args...)
}
