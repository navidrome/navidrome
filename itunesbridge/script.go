package itunesbridge

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

type Script []string

var CommandHost string

func (s Script) lines() []string {
	if len(s) == 0 {
		panic("empty script")
	}

	lines := make([]string, 0, 2)
	tell := `tell application "iTunes"`
	if CommandHost != "" {
		tell += fmt.Sprintf(` of machine %q`, CommandHost)
	}
	if len(s) == 1 {
		tell += " to " + s[0]
		lines = append(lines, tell)
	} else {
		lines = append(lines, tell)
		lines = append(lines, s...)
		lines = append(lines, "end tell")
	}
	return lines
}

func (s Script) args() []string {
	var args []string
	for _, line := range s.lines() {
		args = append(args, "-e", line)
	}
	return args
}

func (s Script) Command(w io.Writer, args ...string) *exec.Cmd {
	command := exec.Command("osascript", append(s.args(), args...)...)
	command.Stdout = w
	command.Stderr = os.Stderr
	return command
}

func (s Script) Run(args ...string) error {
	return s.Command(os.Stdout, args...).Run()
}

func (s Script) Output(args ...string) ([]byte, error) {
	return s.Command(nil, args...).Output()
}

func (s Script) OutputString(args ...string) (string, error) {
	p, err := s.Output(args...)
	str := string(p)
	return str, err
}
