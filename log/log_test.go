package log

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	. "github.com/smartystreets/goconvey/convey"
)

func TestLog(t *testing.T) {

	Convey("Test Logger", t, func() {
		l, hook := test.NewNullLogger()
		SetLevel(LevelInfo)
		SetDefaultLogger(l)

		Convey("Plain message", func() {
			Error("Simple Message")
			So(hook.LastEntry().Message, ShouldEqual, "Simple Message")
			So(hook.LastEntry().Data, ShouldBeEmpty)
		})

		Convey("Passing nil as context", func() {
			Error(nil, "Simple Message")
			So(hook.LastEntry().Message, ShouldEqual, "Simple Message")
			So(hook.LastEntry().Data, ShouldBeEmpty)
		})

		SkipConvey("Empty context", func() {
			Error(context.Background(), "Simple Message")
			So(hook.LastEntry().Message, ShouldEqual, "Simple Message")
			So(hook.LastEntry().Data, ShouldBeEmpty)
		})

		Convey("Message with two kv pairs", func() {
			Error("Simple Message", "key1", "value1", "key2", "value2")
			So(hook.LastEntry().Message, ShouldEqual, "Simple Message")
			So(hook.LastEntry().Data["key1"], ShouldEqual, "value1")
			So(hook.LastEntry().Data["key2"], ShouldEqual, "value2")
			So(hook.LastEntry().Data, ShouldHaveLength, 2)
		})

		Convey("Only error", func() {
			Error(errors.New("error test"))
			So(hook.LastEntry().Message, ShouldEqual, "error test")
			So(hook.LastEntry().Data, ShouldBeEmpty)
		})

		Convey("Error as last argument", func() {
			Error("Error scrobbling track", "id", 1, errors.New("some issue"))
			So(hook.LastEntry().Message, ShouldEqual, "Error scrobbling track")
			So(hook.LastEntry().Data["id"], ShouldEqual, 1)
			So(hook.LastEntry().Data["error"], ShouldEqual, "some issue")
			So(hook.LastEntry().Data, ShouldHaveLength, 2)
		})

		Convey("Passing a request", func() {
			ctx := NewContext(nil, "foo", "bar")
			req := httptest.NewRequest("get", "/", nil).WithContext(ctx)

			Error(req, "Simple Message", "key1", "value1")
			So(hook.LastEntry().Message, ShouldEqual, "Simple Message")
			So(hook.LastEntry().Data["foo"], ShouldEqual, "bar")
			So(hook.LastEntry().Data["key1"], ShouldEqual, "value1")
			So(hook.LastEntry().Data, ShouldHaveLength, 2)
		})

		Convey("Skip if level is lower", func() {
			SetLevel(LevelError)
			Info("Simple Message")
			So(hook.LastEntry(), ShouldBeNil)
		})
	})

	Convey("Test extractLogger", t, func() {
		Convey("It returns an error if the context is nil", func() {
			_, err := extractLogger(nil)
			So(err, ShouldNotBeNil)
		})

		Convey("It returns an error if the context is a string", func() {
			_, err := extractLogger("any msg")
			So(err, ShouldNotBeNil)
		})

		Convey("It returns the logger from context if it has one", func() {
			logger := logrus.NewEntry(logrus.New())
			ctx := context.Background()
			ctx = context.WithValue(ctx, "logger", logger)

			l, err := extractLogger(ctx)
			So(err, ShouldBeNil)
			So(l, ShouldEqual, logger)
		})

		Convey("It returns the logger from request's context if it has one", func() {
			logger := logrus.NewEntry(logrus.New())
			ctx := context.Background()
			ctx = context.WithValue(ctx, "logger", logger)
			req := httptest.NewRequest("get", "/", nil).WithContext(ctx)
			l, err := extractLogger(req)
			So(err, ShouldBeNil)
			So(l, ShouldEqual, logger)
		})
	})
}
