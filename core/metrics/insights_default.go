//go:build !linux && !windows && !darwin

package metrics

import "errors"

func getOSVersion() (string, string) { return "", "" }

func getFilesystemType(_ string) (string, error) { return "", errors.New("not implemented") }
