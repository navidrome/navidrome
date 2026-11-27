package cmd

import (
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("systemdScript template", func() {
	systemdKeys := map[string]bool{
		"Description": true, "Path": true, "Name": true, "Dependencies": true,
		"Arguments": true, "ChRoot": true, "WorkingDirectory": true,
		"UserName": true, "ReloadSignal": true, "PIDFile": true,
		"LogDirectory": true, "OutputFileSupport": true, "LimitNOFILE": true,
		"Restart": true, "SuccessExitStatus": true, "EnvVars": true,
	}
	systemdFuncs := map[string]bool{"cmd": true, "cmdEscape": true}

	actionRe := regexp.MustCompile(`\{\{(.*?)\}\}`)

	parseAction := func(action string) (key string, funcs []string) {
		action = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(action, "-"), "-"))
		kw, rest, _ := strings.Cut(action, " ")
		switch kw {
		case "end", "else":
			return "", nil
		case "if", "range":
			return strings.TrimSpace(rest), nil
		}
		parts := strings.Split(action, "|")
		for _, p := range parts[1:] {
			funcs = append(funcs, strings.TrimSpace(p))
		}
		return strings.TrimSpace(parts[0]), funcs
	}

	It("only references keys and functions the service library provides", func() {
		matches := actionRe.FindAllStringSubmatch(systemdScript, -1)
		Expect(matches).ToNot(BeEmpty())

		for _, m := range matches {
			key, funcs := parseAction(m[1])
			if key != "" && key != "." {
				Expect(systemdKeys).To(HaveKey(key),
					"template action %q uses a key unknown to kardianos/service", m[0])
			}
			for _, fn := range funcs {
				Expect(systemdFuncs).To(HaveKey(fn),
					"template action %q uses an unknown pipeline function", m[0])
			}
		}
	})
})
