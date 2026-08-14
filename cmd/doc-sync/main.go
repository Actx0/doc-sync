// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Actx0/doc-sync/internal/core"
	"github.com/samber/lo"
)

var version = "dev"

type stringList []string

func (s *stringList) String() string {
	return strings.Join(*s, ",")
}

func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func firstEnv(keys ...string) string {
	return lo.FirstOrEmpty(lo.FilterMap(keys, func(key string, _ int) (string, bool) {
		value := strings.TrimSpace(os.Getenv(key))
		return value, value != ""
	}))
}

func parseBool(raw string) bool {
	return lo.Contains([]string{"1", "true", "yes", "y"}, strings.ToLower(strings.TrimSpace(raw)))
}

func writeGitHubOutput(path string, result *core.Result) error {
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

func writeStepSummary(path string, result *core.Result) error {
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

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var pathFlags stringList
	cfg := core.Config{}

	flag.StringVar(&cfg.WorkspaceID, "workspace-id", firstEnv("INPUT_WORKSPACE_ID", "ACTX0_WORKSPACE_ID"), "Actx0 workspace id")
	flag.StringVar(&cfg.AccessKey, "access-key", firstEnv("INPUT_ACCESS_KEY", "ACTX0_ACCESS_KEY"), "Actx0 access key")
	flag.StringVar(&cfg.Tags, "tags", firstEnv("INPUT_TAGS", "ACTX0_TAGS"), "knowledge labels as a YAML map (key: value)")
	flag.StringVar(&cfg.RepoDir, "repo-dir", firstEnv("INPUT_REPO_DIR", "ACTX0_REPO_DIR"), "repository root used to resolve paths")
	flag.StringVar(&cfg.BaseURL, "base-url", firstEnv("INPUT_BASE_URL", "ACTX0_BASE_URL"), "Actx0 API base URL")
	flag.BoolVar(&cfg.DryRun, "dry-run", parseBool(firstEnv("INPUT_DRY_RUN", "ACTX0_DRY_RUN")), "compare checksums without uploading")
	pathsRaw := flag.String("paths", firstEnv("INPUT_PATHS", "ACTX0_PATHS"), "files, directories, or globs, comma or newline separated")
	flag.Var(&pathFlags, "path", "file, directory, or glob to sync (repeatable)")
	showVersion := flag.Bool("version", false, "print version")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return nil
	}

	if cfg.RepoDir == "" {
		cfg.RepoDir = "."
	}
	cfg.Paths = append(core.SplitList(*pathsRaw), pathFlags...)
	cfg.Paths = append(cfg.Paths, flag.Args()...)

	result, err := core.Run(context.Background(), cfg, os.Stdout)
	if writeErr := writeGitHubOutput(os.Getenv("GITHUB_OUTPUT"), result); writeErr != nil {
		return writeErr
	}
	if writeErr := writeStepSummary(os.Getenv("GITHUB_STEP_SUMMARY"), result); writeErr != nil {
		return writeErr
	}
	if err != nil {
		return err
	}
	if result.Failed > 0 {
		return fmt.Errorf("failed to sync %d document(s)", result.Failed)
	}
	return nil
}
