package log

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// journalFormatter wraps a logrus.Formatter and prepends a syslog priority
// prefix (<N>) to each log line. When stderr is captured by systemd-journald,
// this prefix tells journald the correct severity for each message.
//
// See https://www.freedesktop.org/software/systemd/man/sd-daemon.html
type journalFormatter struct {
	inner logrus.Formatter
}

// levelToPriority maps logrus levels to syslog priority values.
// The mapping follows RFC 5424 severity levels.
var levelToPriority = map[logrus.Level]int{
	logrus.PanicLevel: 0, // emerg
	logrus.FatalLevel: 2, // crit
	logrus.ErrorLevel: 3, // err
	logrus.WarnLevel:  4, // warning
	logrus.InfoLevel:  6, // info
	logrus.DebugLevel: 7, // debug
	logrus.TraceLevel: 7, // debug
}

func (f *journalFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	formatted, err := f.inner.Format(entry)
	if err != nil {
		return formatted, err
	}
	priority, ok := levelToPriority[entry.Level]
	if !ok {
		priority = 6 // default to info for unknown levels
	}
	prefix := []byte(fmt.Sprintf("<%d>", priority))
	return append(prefix, formatted...), nil
}
