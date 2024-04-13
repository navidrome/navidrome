package log

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type Level uint8

type LevelFunc = func(ctx interface{}, msg interface{}, keyValuePairs ...interface{})

var redacted = &Hook{
	AcceptedLevels: logrus.AllLevels,
	RedactionList: []string{
		// Keys from the config
		"(ApiKey:\")[\\w]*",
		"(Secret:\")[\\w]*",
		"(Spotify.*ID:\")[\\w]*",
		"(PasswordEncryptionKey:[\\s]*\")[^\"]*",
		"(ReverseProxyUserHeader:[\\s]*\")[^\"]*",
		"(ReverseProxyWhitelist:[\\s]*\")[^\"]*",
		"(MetricsPath:[\\s]*\")[^\"]*",
		"(DevAutoCreateAdminPassword:[\\s]*\")[^\"]*",
		"(DevAutoLoginUsername:[\\s]*\")[^\"]*",

		// UI appConfig
		"(subsonicToken:)[\\w]+(\\s)",
		"(subsonicSalt:)[\\w]+(\\s)",
		"(token:)[^\\s]+",

		// Subsonic query params
		"([^\\w]t=)[\\w]+",
		"([^\\w]s=)[^&]+",
		"([^\\w]p=)[^&]+",
		"([^\\w]jwt=)[^&]+",

		// External services query params
		"([^\\w]api_key=)[\\w]+",
	},
}

const (
	LevelFatal = Level(logrus.FatalLevel)
	LevelError = Level(logrus.ErrorLevel)
	LevelWarn  = Level(logrus.WarnLevel)
	LevelInfo  = Level(logrus.InfoLevel)
	LevelDebug = Level(logrus.DebugLevel)
	LevelTrace = Level(logrus.TraceLevel)
)

type contextKey string

const loggerCtxKey = contextKey("logger")

type levelPath struct {
	path  string
	level Level
}

var (
	currentLevel  Level
	defaultLogger = logrus.New()
	logSourceLine = false
	rootPath      string
	logLevels     []levelPath
)

// SetLevel sets the global log level used by the simple logger.
func SetLevel(l Level) {
	currentLevel = l
	defaultLogger.Level = logrus.TraceLevel
	logrus.SetLevel(logrus.Level(l))
}

func SetLevelString(l string) {
	level := levelFromString(l)
	SetLevel(level)
}

func levelFromString(l string) Level {
	envLevel := strings.ToLower(l)
	var level Level
	switch envLevel {
	case "fatal":
		level = LevelFatal
	case "error":
		level = LevelError
	case "warn":
		level = LevelWarn
	case "debug":
		level = LevelDebug
	case "trace":
		level = LevelTrace
	default:
		level = LevelInfo
	}
	return level
}

// SetLogLevels sets the log levels for specific paths in the codebase.
func SetLogLevels(levels map[string]string) {
	for k, v := range levels {
		logLevels = append(logLevels, levelPath{path: k, level: levelFromString(v)})
	}
	sort.Slice(logLevels, func(i, j int) bool {
		return logLevels[i].path > logLevels[j].path
	})
}

func SetLogSourceLine(enabled bool) {
	logSourceLine = enabled
}

func SetRedacting(enabled bool) {
	if enabled {
		defaultLogger.AddHook(redacted)
	}
}

// Redact applies redaction to a single string
func Redact(msg string) string {
	r, _ := redacted.redact(msg)
	return r
}

func NewContext(ctx context.Context, keyValuePairs ...interface{}) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	logger, ok := ctx.Value(loggerCtxKey).(*logrus.Entry)
	if !ok {
		logger = createNewLogger()
	}
	logger = addFields(logger, keyValuePairs)
	ctx = context.WithValue(ctx, loggerCtxKey, logger)

	return ctx
}

func SetDefaultLogger(l *logrus.Logger) {
	defaultLogger = l
}

func CurrentLevel() Level {
	return currentLevel
}

// IsGreaterOrEqualTo returns true if the caller's current log level is equal or greater than the provided level.
func IsGreaterOrEqualTo(level Level) bool {
	return shouldLog(level)
}

func Fatal(args ...interface{}) {
	log(LevelFatal, args...)
	os.Exit(1)
}

func Error(args ...interface{}) {
	log(LevelError, args...)
}

func Warn(args ...interface{}) {
	log(LevelWarn, args...)
}

func Info(args ...interface{}) {
	log(LevelInfo, args...)
}

func Debug(args ...interface{}) {
	log(LevelDebug, args...)
}

func Trace(args ...interface{}) {
	log(LevelTrace, args...)
}

func log(level Level, args ...interface{}) {
	if !shouldLog(level) {
		return
	}
	logger, msg := parseArgs(args)
	logger.Log(logrus.Level(level), msg)
}

func shouldLog(requiredLevel Level) bool {
	if currentLevel >= requiredLevel {
		return true
	}
	if len(logLevels) == 0 {
		return false
	}

	_, file, _, ok := runtime.Caller(3)
	if !ok {
		return false
	}

	file = strings.TrimPrefix(file, rootPath)
	for _, lp := range logLevels {
		if strings.HasPrefix(file, lp.path) {
			return lp.level >= requiredLevel
		}
	}
	return false
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
	if logSourceLine {
		_, file, line, ok := runtime.Caller(3)
		if !ok {
			file = "???"
			line = 0
		}
		//_, filename := path.Split(file)
		//l = l.WithField("filename", filename).WithField("line", line)
		l = l.WithField(" source", fmt.Sprintf("file://%s:%d", file, line))
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
			if i+1 >= len(keyValuePairs) {
				logger = logger.WithField(name, "!!!!Invalid number of arguments in log call!!!!")
			} else {
				switch v := keyValuePairs[i+1].(type) {
				case time.Duration:
					logger = logger.WithField(name, ShortDur(v))
				case fmt.Stringer:
					vOf := reflect.ValueOf(v)
					if vOf.Kind() == reflect.Pointer && vOf.IsNil() {
						logger = logger.WithField(name, "nil")
					} else {
						logger = logger.WithField(name, v.String())
					}
				default:
					logger = logger.WithField(name, v)
				}
			}
		}
	}
	return logger
}

func extractLogger(ctx interface{}) (*logrus.Entry, error) {
	switch ctx := ctx.(type) {
	case *logrus.Entry:
		return ctx, nil
	case context.Context:
		logger := ctx.Value(loggerCtxKey)
		if logger != nil {
			return logger.(*logrus.Entry), nil
		}
		return extractLogger(NewContext(ctx))
	case *http.Request:
		return extractLogger(ctx.Context())
	}
	return nil, errors.New("no logger found")
}

func createNewLogger() *logrus.Entry {
	//logrus.SetFormatter(&logrus.TextFormatter{ForceColors: true, DisableTimestamp: false, FullTimestamp: true})
	//l.Formatter = &logrus.TextFormatter{ForceColors: true, DisableTimestamp: false, FullTimestamp: true}
	logger := logrus.NewEntry(defaultLogger)
	return logger
}

func init() {
	defaultLogger.Level = logrus.TraceLevel
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return
	}
	rootPath = strings.TrimSuffix(file, "log/log.go")
}
