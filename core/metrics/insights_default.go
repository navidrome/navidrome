//go:build !linux && !windows && !darwin

package metrics

func getOSVersion() (string, string) {
	return "", ""
}
