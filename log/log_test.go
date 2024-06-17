package log

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

func TestLog(t *testing.T) {
	SetLevel(LevelInfo)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Log Suite")
}

var _ = Describe("Logger", func() {
	var l *logrus.Logger
	var hook *test.Hook

	BeforeEach(func() {
		l, hook = test.NewNullLogger()
		SetLevel(LevelTrace)
		SetDefaultLogger(l)
	})

	Describe("Logging", func() {
		It("logs a simple message", func() {
			Error("Simple Message")
			Expect(hook.LastEntry().Message).To(Equal("Simple Message"))
			Expect(hook.LastEntry().Data).To(BeEmpty())
		})

		It("logs a message when context is nil", func() {
			Error(nil, "Simple Message")
			Expect(hook.LastEntry().Message).To(Equal("Simple Message"))
			Expect(hook.LastEntry().Data).To(BeEmpty())
		})

		It("Empty context", func() {
			Error(context.TODO(), "Simple Message")
			Expect(hook.LastEntry().Message).To(Equal("Simple Message"))
			Expect(hook.LastEntry().Data).To(BeEmpty())
		})

		It("logs messages with two kv pairs", func() {
			Error("Simple Message", "key1", "value1", "key2", "value2")
			Expect(hook.LastEntry().Message).To(Equal("Simple Message"))
			Expect(hook.LastEntry().Data["key1"]).To(Equal("value1"))
			Expect(hook.LastEntry().Data["key2"]).To(Equal("value2"))
			Expect(hook.LastEntry().Data).To(HaveLen(2))
		})

		It("logs error objects as simple messages", func() {
			Error(errors.New("error test"))
			Expect(hook.LastEntry().Message).To(Equal("error test"))
			Expect(hook.LastEntry().Data).To(BeEmpty())
		})

		It("logs errors passed as last argument", func() {
			Error("Error scrobbling track", "id", 1, errors.New("some issue"))
			Expect(hook.LastEntry().Message).To(Equal("Error scrobbling track"))
			Expect(hook.LastEntry().Data["id"]).To(Equal(1))
			Expect(hook.LastEntry().Data["error"]).To(Equal("some issue"))
			Expect(hook.LastEntry().Data).To(HaveLen(2))
		})

		It("can get data from the request's context", func() {
			ctx := NewContext(context.TODO(), "foo", "bar")
			req := httptest.NewRequest("get", "/", nil).WithContext(ctx)

			Error(req, "Simple Message", "key1", "value1")

			Expect(hook.LastEntry().Message).To(Equal("Simple Message"))
			Expect(hook.LastEntry().Data["foo"]).To(Equal("bar"))
			Expect(hook.LastEntry().Data["key1"]).To(Equal("value1"))
			Expect(hook.LastEntry().Data).To(HaveLen(2))
		})

		It("does not log anything if level is lower", func() {
			SetLevel(LevelError)
			Info("Simple Message")
			Expect(hook.LastEntry()).To(BeNil())
		})

		It("logs source file and line number, if requested", func() {
			SetLogSourceLine(true)
			Error("A crash happened")
			// NOTE: This assertion breaks if the line number above changes
			Expect(hook.LastEntry().Data[" source"]).To(ContainSubstring("/log/log_test.go:93"))
			Expect(hook.LastEntry().Message).To(Equal("A crash happened"))
		})

		It("logs fmt.Stringer as a string", func() {
			t := time.Now()
			Error("Simple Message", "key1", t)
			Expect(hook.LastEntry().Data["key1"]).To(Equal(t.String()))
		})
		It("logs nil fmt.Stringer as nil", func() {
			var t *time.Time
			Error("Simple Message", "key1", t)
			Expect(hook.LastEntry().Data["key1"]).To(Equal("nil"))
		})
	})

	Describe("Levels", func() {
		BeforeEach(func() {
			SetLevel(LevelTrace)
		})
		It("logs error messages", func() {
			Error("msg")
			Expect(hook.LastEntry().Level).To(Equal(logrus.ErrorLevel))
		})
		It("logs warn messages", func() {
			Warn("msg")
			Expect(hook.LastEntry().Level).To(Equal(logrus.WarnLevel))
		})
		It("logs info messages", func() {
			Info("msg")
			Expect(hook.LastEntry().Level).To(Equal(logrus.InfoLevel))
		})
		It("logs debug messages", func() {
			Debug("msg")
			Expect(hook.LastEntry().Level).To(Equal(logrus.DebugLevel))
		})
		It("logs trace messages", func() {
			Trace("msg")
			Expect(hook.LastEntry().Level).To(Equal(logrus.TraceLevel))
		})
	})

	Describe("LogLevels", func() {
		It("logs at specific levels", func() {
			SetLevel(LevelError)
			Debug("message 1")
			Expect(hook.LastEntry()).To(BeNil())

			SetLogLevels(map[string]string{
				"log/log_test": "debug",
			})

			Debug("message 2")
			Expect(hook.LastEntry().Message).To(Equal("message 2"))
		})
	})

	Describe("IsGreaterOrEqualTo", func() {
		BeforeEach(func() {
			SetLogLevels(nil)
		})

		It("returns false if log level is below provided level", func() {
			SetLevel(LevelError)
			Expect(IsGreaterOrEqualTo(LevelWarn)).To(BeFalse())
		})

		It("returns true if log level is equal to provided level", func() {
			SetLevel(LevelWarn)
			Expect(IsGreaterOrEqualTo(LevelWarn)).To(BeTrue())
		})

		It("returns true if log level is above provided level", func() {
			SetLevel(LevelTrace)
			Expect(IsGreaterOrEqualTo(LevelDebug)).To(BeTrue())
		})

		It("returns true if log level for the current code path is equal provided level", func() {
			SetLevel(LevelError)
			SetLogLevels(map[string]string{
				"log/log_test": "debug",
			})

			// Need to nest it in a function to get the correct code path
			var result = func() bool {
				return IsGreaterOrEqualTo(LevelDebug)
			}()

			Expect(result).To(BeTrue())
		})
	})

	Describe("extractLogger", func() {
		It("returns an error if the context is nil", func() {
			_, err := extractLogger(nil)
			Expect(err).ToNot(BeNil())
		})

		It("returns an error if the context is a string", func() {
			_, err := extractLogger("any msg")
			Expect(err).ToNot(BeNil())
		})

		It("returns the logger from context if it has one", func() {
			logger := logrus.NewEntry(logrus.New())
			ctx := context.Background()
			ctx = context.WithValue(ctx, loggerCtxKey, logger)

			Expect(extractLogger(ctx)).To(Equal(logger))
		})

		It("returns the logger from request's context if it has one", func() {
			logger := logrus.NewEntry(logrus.New())
			ctx := context.Background()
			ctx = context.WithValue(ctx, loggerCtxKey, logger)
			req := httptest.NewRequest("get", "/", nil).WithContext(ctx)

			Expect(extractLogger(req)).To(Equal(logger))
		})
	})

	Describe("SetLevelString", func() {
		It("converts Fatal level", func() {
			SetLevelString("Fatal")
			Expect(CurrentLevel()).To(Equal(LevelFatal))
		})
		It("converts Error level", func() {
			SetLevelString("ERROR")
			Expect(CurrentLevel()).To(Equal(LevelError))
		})
		It("converts Warn level", func() {
			SetLevelString("warn")
			Expect(CurrentLevel()).To(Equal(LevelWarn))
		})
		It("converts Info level", func() {
			SetLevelString("info")
			Expect(CurrentLevel()).To(Equal(LevelInfo))
		})
		It("converts Debug level", func() {
			SetLevelString("debug")
			Expect(CurrentLevel()).To(Equal(LevelDebug))
		})
		It("converts Trace level", func() {
			SetLevelString("trace")
			Expect(CurrentLevel()).To(Equal(LevelTrace))
		})
	})

	Describe("Redact", func() {
		Describe("Subsonic API password", func() {
			msg := "getLyrics.view?v=1.2.0&c=iSub&u=user_name&p=first%20and%20other%20words&title=Title"
			Expect(Redact(msg)).To(Equal("getLyrics.view?v=1.2.0&c=iSub&u=user_name&p=[REDACTED]&title=Title"))
		})
	})
})
