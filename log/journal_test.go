package log

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("journalFormatter", func() {
	var formatter *journalFormatter

	BeforeEach(func() {
		inner := &logrus.TextFormatter{
			DisableTimestamp: true,
			DisableColors:    true,
		}
		formatter = &journalFormatter{inner: inner}
	})

	DescribeTable("prefixes log lines with syslog priority",
		func(level logrus.Level, expectedPrefix string) {
			entry := &logrus.Entry{
				Logger:  logrus.New(),
				Level:   level,
				Message: "test message",
				Data:    logrus.Fields{},
			}
			out, err := formatter.Format(entry)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(out)).To(HavePrefix(expectedPrefix))
		},
		Entry("error", logrus.ErrorLevel, "<3>"),
		Entry("warning", logrus.WarnLevel, "<4>"),
		Entry("info", logrus.InfoLevel, "<6>"),
		Entry("debug", logrus.DebugLevel, "<7>"),
		Entry("trace", logrus.TraceLevel, "<7>"),
		Entry("fatal", logrus.FatalLevel, "<2>"),
		Entry("panic", logrus.PanicLevel, "<0>"),
		Entry("unknown level defaults to info", logrus.Level(99), "<6>"),
	)
})
