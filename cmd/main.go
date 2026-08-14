// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/Actx0/doc-sync/core"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var pathFlags core.StringList
	cfg := core.Config{}

	flag.StringVar(&cfg.WorkspaceID, "workspace-id", core.FirstEnv("INPUT_WORKSPACE_ID", "ACTX0_WORKSPACE_ID"), "Actx0 workspace id")
	flag.StringVar(&cfg.AccessKey, "access-key", core.FirstEnv("INPUT_ACCESS_KEY", "ACTX0_ACCESS_KEY"), "Actx0 access key")
	flag.StringVar(&cfg.Tags, "tags", core.FirstEnv("INPUT_TAGS", "ACTX0_TAGS"), "knowledge labels as a YAML map (key: value)")
	flag.StringVar(&cfg.RepoDir, "repo-dir", core.FirstEnv("INPUT_REPO_DIR", "ACTX0_REPO_DIR"), "repository root used to resolve paths")
	flag.StringVar(&cfg.BaseURL, "base-url", core.FirstEnv("INPUT_BASE_URL", "ACTX0_BASE_URL"), "Actx0 API base URL")
	flag.BoolVar(&cfg.DryRun, "dry-run", core.ParseBool(core.FirstEnv("INPUT_DRY_RUN", "ACTX0_DRY_RUN")), "compare checksums without uploading")
	pathsRaw := flag.String("paths", core.FirstEnv("INPUT_PATHS", "ACTX0_PATHS"), "files, directories, or globs, comma or newline separated")
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

	if err := core.WriteGitHubOutput(os.Getenv("GITHUB_OUTPUT"), result); err != nil {
		return err
	}

	if err := core.WriteStepSummary(os.Getenv("GITHUB_STEP_SUMMARY"), result); err != nil {
		return err
	}

	if err != nil {
		return err
	}
	if result.Failed > 0 {
		return fmt.Errorf("failed to sync %d document(s)", result.Failed)
	}

	return nil
}
