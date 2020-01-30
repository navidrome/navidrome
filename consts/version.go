package consts

import "fmt"

var (
	// This will be set in build time. If not, version will be set to "dev"
	gitTag string
	gitSha string
)

// Formats:
// dev
// v0.2.0 (5b84188)
// master (9ed35cb)
func Version() string {
	if gitSha == "" {
		return "dev"
	}
	return fmt.Sprintf("%s (%s)", gitTag, gitSha)
}
