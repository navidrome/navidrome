//go:build !linux && !windows && !darwin

package metrics

import (
	"context"
	"errors"
)

func getOSVersion(context.Context) (string, string) { return "", "" }

func getFilesystemType(_ string) (string, error) { return "", errors.New("not implemented") }
