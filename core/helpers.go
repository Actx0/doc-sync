// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package core

import (
	"fmt"
	"os"
	"strings"

	"github.com/samber/lo"
)

type StringList []string

func (s *StringList) String() string {
	return strings.Join(*s, ",")
}

func (s *StringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func FirstEnv(keys ...string) string {
	return lo.FirstOrEmpty(lo.FilterMap(keys, func(key string, _ int) (string, bool) {
		value := strings.TrimSpace(os.Getenv(key))
		return value, value != ""
	}))
}

func ParseBool(raw string) bool {
	return lo.Contains([]string{"1", "true", "yes", "y"}, strings.ToLower(strings.TrimSpace(raw)))
}

func WriteGitHubOutput(path string, result *Result) error {
	if path == "" || result == nil {
		return nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "uploaded=%d\nreplaced=%d\nskipped=%d\nfailed=%d\n",
		result.Uploaded, result.Replaced, result.Skipped, result.Failed)
	return err
}

func WriteStepSummary(path string, result *Result) error {
	if path == "" || result == nil {
		return nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, `## Actx0 knowledge sync

| Uploaded | Replaced | Skipped | Failed |
| --- | --- | --- | --- |
| %d | %d | %d | %d |
`, result.Uploaded, result.Replaced, result.Skipped, result.Failed)
	return err
}
