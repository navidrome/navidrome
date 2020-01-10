package log

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

type Level uint8

type LevelFunc = func(ctx interface{}, msg interface{}, keyValuePairs ...interface{})

const (
	LevelCritical = Level(logrus.FatalLevel)
	LevelError    = Level(logrus.ErrorLevel)
	LevelWarn     = Level(logrus.WarnLevel)
	LevelInfo     = Level(logrus.InfoLevel)
	LevelDebug    = Level(logrus.DebugLevel)
	LevelTrace    = Level(logrus.TraceLevel)
)

var (
	currentLevel  Level
	defaultLogger = logrus.New()
)

// SetLevel sets the global log level used by the simple logger.
func SetLevel(l Level) {
	currentLevel = l
	logrus.SetLevel(logrus.Level(l))
}

func NewContext(ctx context.Context, keyValuePairs ...interface{}) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	logger := addFields(createNewLogger(), keyValuePairs)
	ctx = context.WithValue(ctx, "logger", logger)

	return ctx
}

func SetDefaultLogger(l *logrus.Logger) {
	defaultLogger = l
}

func CurrentLevel() Level {
	return currentLevel
}

func Error(args ...interface{}) {
	if currentLevel < LevelError {
		return
	}
	logger, msg := parseArgs(args)
	logger.Error(msg)
}

func Warn(args ...interface{}) {
	if currentLevel < LevelWarn {
		return
	}
	logger, msg := parseArgs(args)
	logger.Warn(msg)
}

func Info(args ...interface{}) {
	if currentLevel < LevelInfo {
		return
	}
	logger, msg := parseArgs(args)
	logger.Info(msg)
}

func Debug(args ...interface{}) {
	if currentLevel < LevelDebug {
		return
	}
	logger, msg := parseArgs(args)
	logger.Debug(msg)
}

func Trace(args ...interface{}) {
	if currentLevel < LevelTrace {
		return
	}
	logger, msg := parseArgs(args)
	logger.Trace(msg)
}

func parseArgs(args []interface{}) (*logrus.Entry, string) {
	var l *logrus.Entry
	var err error
	if args[0] == nil {
		l = createNewLogger()
		args = args[1:]
	} else {
		l, err = extractLogger(args[0])
		if err != nil {
			l = createNewLogger()
		} else {
			args = args[1:]
		}
	}
	if len(args) > 1 {
		kvPairs := args[1:]
		l = addFields(l, kvPairs)
	}
	switch msg := args[0].(type) {
	case error:
		return l, msg.Error()
	case string:
		return l, msg
	}
	return l, ""
}

func addFields(logger *logrus.Entry, keyValuePairs []interface{}) *logrus.Entry {
	for i := 0; i < len(keyValuePairs); i += 2 {
		switch name := keyValuePairs[i].(type) {
		case error:
			logger = logger.WithField("error", name.Error())
		case string:
			value := keyValuePairs[i+1]
			logger = logger.WithField(name, value)
		}
	}
	return logger
}

func extractLogger(ctx interface{}) (*logrus.Entry, error) {
	switch ctx := ctx.(type) {
	case *logrus.Entry:
		return ctx, nil
	case context.Context:
		logger := ctx.Value("logger")
		if logger != nil {
			return logger.(*logrus.Entry), nil
		}
	case *http.Request:
		return extractLogger(ctx.Context())
	}
	return nil, errors.New("no logger found")
}

func createNewLogger() *logrus.Entry {
	//l.Formatter = &logrus.TextFormatter{ForceColors: true, DisableTimestamp: false, FullTimestamp: true}
	defaultLogger.Level = logrus.Level(currentLevel)
	logger := logrus.NewEntry(defaultLogger)
	logger.Level = logrus.Level(currentLevel)
	return logger
}

func init() {
	//logrus.SetFormatter(&logrus.TextFormatter{ForceColors: true, DisableTimestamp: false, FullTimestamp: true})
	envLevel := strings.ToLower(os.Getenv("LOG_LEVEL"))
	var level Level
	switch envLevel {
	case "critical":
		level = LevelCritical
	case "error":
		level = LevelError
	case "warn":
		level = LevelWarn
	case "debug":
		level = LevelDebug
	default:
		level = LevelInfo
	}
	SetLevel(level)
}
