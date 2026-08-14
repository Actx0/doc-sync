// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package core

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/samber/lo"
)

const DefaultBaseURL = "https://app.actx0.com"

// Config is the input for a knowledge sync run.
type Config struct {
	WorkspaceID string
	AccessKey   string
	Tags        string
	Paths       []string
	RepoDir     string
	BaseURL     string
	DryRun      bool
	Timeout     time.Duration
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.WorkspaceID) == "" {
		return fmt.Errorf("workspace id is required")
	}
	if strings.TrimSpace(c.AccessKey) == "" {
		return fmt.Errorf("access key is required")
	}
	if len(c.Paths) == 0 {
		return fmt.Errorf("at least one path is required")
	}
	if strings.TrimSpace(c.RepoDir) == "" {
		return fmt.Errorf("repo dir is required")
	}
	info, err := os.Stat(c.RepoDir)
	if err != nil {
		return fmt.Errorf("repo dir: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("repo dir is not a directory: %s", c.RepoDir)
	}
	return nil
}

func (c Config) baseURL() string {
	return lo.CoalesceOrEmpty(strings.TrimSpace(c.BaseURL), DefaultBaseURL)
}

func (c Config) timeout() time.Duration {
	return lo.CoalesceOrEmpty(c.Timeout, 2*time.Minute)
}

// ParseLabels parses a YAML-style map of knowledge labels.
//
//	tag: docs
//	team: platform
func ParseLabels(raw string) (map[string]string, error) {
	labels := map[string]string{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "- ")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if !ok || key == "" || value == "" {
			return nil, fmt.Errorf("invalid tag %q, expected key: value", line)
		}
		labels[key] = value
	}
	return labels, nil
}

// SplitList splits newline, comma, or whitespace-separated values.
func SplitList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	replaced := strings.NewReplacer(",", "\n", ";", "\n").Replace(raw)
	return lo.Uniq(lo.FilterMap(strings.Split(replaced, "\n"), func(part string, _ int) (string, bool) {
		part = strings.TrimSpace(part)
		if part == "" || strings.HasPrefix(part, "#") {
			return "", false
		}
		return part, true
	}))
}
