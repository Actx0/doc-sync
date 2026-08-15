// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package core

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/samber/lo"
)

// CollectFiles resolves directories, globs, and files relative to repoDir.
func CollectFiles(repoDir string, patterns []string) ([]string, error) {
	repoDir, err := filepath.Abs(repoDir)
	if err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	var files []string
	add := func(path string) error {
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		info, err := os.Stat(abs)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(repoDir, abs)
		if err != nil {
			return err
		}
		if lo.SomeBy(strings.Split(filepath.ToSlash(rel), "/"), func(part string) bool {
			return strings.HasPrefix(part, ".") || lo.Contains([]string{"node_modules", "vendor"}, part)
		}) {
			return nil
		}
		if _, ok := seen[abs]; ok {
			return nil
		}
		seen[abs] = struct{}{}
		files = append(files, abs)
		return nil
	}

	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" || strings.HasPrefix(pattern, "#") {
			continue
		}
		pattern = filepath.ToSlash(pattern)
		if strings.ContainsAny(pattern, "*?[") {
			matches, err := doublestar.FilepathGlob(filepath.Join(repoDir, filepath.FromSlash(pattern)))
			if err != nil {
				return nil, fmt.Errorf("%s: %w", pattern, err)
			}
			if len(matches) == 0 {
				return nil, fmt.Errorf("no files matched %s", pattern)
			}
			for _, match := range matches {
				if err := add(match); err != nil {
					return nil, err
				}
			}
			continue
		}

		target := pattern
		if !filepath.IsAbs(target) {
			target = filepath.Join(repoDir, filepath.FromSlash(pattern))
		}
		info, err := os.Stat(target)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", pattern, err)
		}
		if info.IsDir() {
			err := filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					if path != target && (strings.HasPrefix(d.Name(), ".") || lo.Contains([]string{"node_modules", "vendor"}, d.Name())) {
						return filepath.SkipDir
					}
					return nil
				}
				if !lo.Contains([]string{".md", ".mdx", ".markdown", ".txt"}, strings.ToLower(filepath.Ext(path))) {
					return nil
				}
				return add(path)
			})
			if err != nil {
				return nil, fmt.Errorf("%s: %w", pattern, err)
			}
			continue
		}
		if err := add(target); err != nil {
			return nil, err
		}
	}

	sort.Strings(files)
	return files, nil
}

func RelFilename(repoDir, absPath string) (string, error) {
	rel, err := filepath.Rel(repoDir, absPath)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}

// DocumentName is the filename Actx0 stores (basename only).
func DocumentName(filename string) string {
	return filepath.ToSlash(filepath.Base(filename))
}

func TitleFromFilename(filename string) string {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	name = strings.ReplaceAll(name, "\\", "/")
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	return strings.TrimSpace(name)
}

func uniqueBasenames(repoDir string, files []string) error {
	grouped := map[string][]string{}
	for _, abs := range files {
		rel, err := RelFilename(repoDir, abs)
		if err != nil {
			rel = abs
		}
		name := DocumentName(rel)
		grouped[name] = append(grouped[name], rel)
	}

	var conflicts []string
	for name, paths := range grouped {
		if len(paths) < 2 {
			continue
		}
		sort.Strings(paths)
		conflicts = append(conflicts, fmt.Sprintf("%s (%s)", name, strings.Join(paths, ", ")))
	}
	if len(conflicts) == 0 {
		return nil
	}
	sort.Strings(conflicts)
	return fmt.Errorf("duplicate filenames: %s", strings.Join(conflicts, "; "))
}

func ContentType(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".md", ".mdx", ".markdown":
		return "text/markdown"
	case ".txt":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}
