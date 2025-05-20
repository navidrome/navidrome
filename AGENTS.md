# Testing Instructions

- **No implementation task is considered complete until it includes thorough, passing tests that cover the new or
  changed functionality. All new code must be accompanied by Ginkgo/Gomega tests, and PRs/commits without tests should
  be considered incomplete.**
- All Go tests in this project **MUST** be written using the **Ginkgo v2** and **Gomega** frameworks.
- To run all tests, use `make test`.
- To run tests for a specific package, use `make test PKG=./pkgname/...`
- Do not run tests in parallel
- Don't use `--fail-on-pending`

## Mocking Convention

- Always try to use the mocks provided in the `tests` package before creating a new mock implementation.
- Only create a new mock if the required functionality is not covered by the existing mocks in `tests`.
- Never mock a real implementation when testing. Remember: there is no value in testing an interface, only the real implementation.

## Example

Every package that you write tests for, should have a `*_suite_test.go` file, to hook up the Ginkgo test suite. Example:
```
package core

import (
	"testing"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCore(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Core Suite")
}
```
Never put a `func Test*` in regular *_test.go files, only in `*_suite_test.go` files.

Refer to existing test suites for examples of proper setup and usage, such as the one defined in @core_suite_test.go

## Exceptions

There should be no exceptions to this rule. If you encounter tests written with the standard `testing` package or other frameworks, they should be refactored to use Ginkgo/Gomega. If you need a new mock, first confirm that it does not already exist in the `tests` package.

### Configuration

You can set config values in the BeforeEach/BeforeAll blocks. If you do so, remember to add `DeferCleanup(configtest.SetupConfig())` to reset the values. Example:

```go
BeforeEach(func() {
    DeferCleanup(configtest.SetupConfig())
    conf.Server.EnableDownloads = true
})
```

# Logging System Usage Guide

This project uses a custom logging system built on top of logrus, `log/log.go`. Follow these conventions for all logging:

## Logging API
- Use the provided functions for logging at different levels:
    - `Error(...)`, `Warn(...)`, `Info(...)`, `Debug(...)`, `Trace(...)`, `Fatal(...)`
- These functions accept flexible arguments:
    - The first argument can be a context (`context.Context`), an HTTP request, or `nil`.
    - The next argument is the log message (string or error).
    - Additional arguments are key-value pairs (e.g., `"key", value`).
    - If the last argument is an error, it is logged under the `error` key.

**Examples:**
```go
log.Error("A message")
log.Error(ctx, "A message with context")
log.Error("Failed to save", "id", 123, err)
log.Info(req, "Request received", "user", userID)
```

## Logging errors
- You don't need to add "err" key when logging an error, it is automatically added.
- Error must always be the last parameter in the log call.
  Examples:
```go
log.Error("Failed to save", "id", 123, err) // GOOD
log.Error("Failed to save", "id", 123, "err", err) // BAD
log.Error("Failed to save", err, "id", 123) // BAD
```

## Context and Request Logging
- If a context or HTTP request is passed as the first argument, any logger fields in the context are included in the log entry.
- Use `log.NewContext(ctx, "key", value, ...)` to add fields to a context for logging.

## Log Levels
- Set the global log level with `log.SetLevel(log.LevelInfo)` or `log.SetLevelString("info")`.
- Per-path log levels can be set with `log.SetLogLevels(map[string]string{"path": "level"})`.
- Use `log.IsGreaterOrEqualTo(level)` to check if a log level is enabled for the current code path.

## Source Line Logging
- Enable source file/line logging with `log.SetLogSourceLine(true)`.

## Best Practices
- Always use the logging API, never log directly with logrus or fmt.
- Prefer structured logging (key-value pairs) for important data.
- Use context/request logging for traceability in web handlers.
- For tests, use Ginkgo/Gomega and set up a test logger as in `log/log_test.go`.

## See Also
- `log/log.go` for implementation details
- `log/log_test.go` for usage examples and test patterns
