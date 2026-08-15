// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLabels(t *testing.T) {
	t.Run("yaml map", func(t *testing.T) {
		got, err := ParseLabels("tag: docs\nteam: platform\nrepo: actx0/app")
		require.NoError(t, err)
		assert.Equal(t, map[string]string{
			"tag":  "docs",
			"team": "platform",
			"repo": "actx0/app",
		}, got)
	})

	t.Run("list style dashes", func(t *testing.T) {
		got, err := ParseLabels("- tag: docs\n- team: platform")
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"tag": "docs", "team": "platform"}, got)
	})

	t.Run("invalid tag", func(t *testing.T) {
		_, err := ParseLabels("platform")
		require.Error(t, err)
	})
}

func TestSplitList(t *testing.T) {
	got := SplitList("docs/\ndocs/*.md,docs/dd.md\n# ignore\n docs/ree.md ")
	assert.Equal(t, []string{"docs/", "docs/*.md", "docs/dd.md", "docs/ree.md"}, got)
}

func TestCollectFiles(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	nested := filepath.Join(docs, "nested")
	require.NoError(t, os.MkdirAll(nested, 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(docs, ".git"), 0o755))

	write := func(rel, body string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	}
	write("docs/dd.md", "dd")
	write("docs/ree.md", "ree")
	write("docs/notes.txt", "notes")
	write("docs/skip.bin", "bin")
	write("docs/nested/more.md", "more")
	write("docs/.git/config.md", "nope")
	write("README.md", "root")

	rel := func(paths []string) []string {
		out := make([]string, len(paths))
		for i, path := range paths {
			name, err := RelFilename(root, path)
			require.NoError(t, err)
			out[i] = name
		}
		return out
	}

	t.Run("directory", func(t *testing.T) {
		files, err := CollectFiles(root, []string{"docs/"})
		require.NoError(t, err)
		assert.Equal(t, []string{"docs/dd.md", "docs/nested/more.md", "docs/notes.txt", "docs/ree.md"}, rel(files))
	})

	t.Run("glob", func(t *testing.T) {
		files, err := CollectFiles(root, []string{"docs/*.md"})
		require.NoError(t, err)
		assert.Equal(t, []string{"docs/dd.md", "docs/ree.md"}, rel(files))
	})

	t.Run("specific files", func(t *testing.T) {
		files, err := CollectFiles(root, []string{"docs/dd.md", "docs/ree.md"})
		require.NoError(t, err)
		assert.Equal(t, []string{"docs/dd.md", "docs/ree.md"}, rel(files))
	})

	t.Run("recursive glob", func(t *testing.T) {
		files, err := CollectFiles(root, []string{"docs/**/*.md"})
		require.NoError(t, err)
		assert.Equal(t, []string{"docs/dd.md", "docs/nested/more.md", "docs/ree.md"}, rel(files))
	})

	t.Run("dedupes overlapping patterns", func(t *testing.T) {
		files, err := CollectFiles(root, []string{"docs/", "docs/*.md", "docs/dd.md", "docs/ree.md"})
		require.NoError(t, err)
		assert.Equal(t, []string{"docs/dd.md", "docs/nested/more.md", "docs/notes.txt", "docs/ree.md"}, rel(files))
	})
}

func TestTitleFromFilename(t *testing.T) {
	assert.Equal(t, "docs/getting started", TitleFromFilename("docs/getting-started.md"))
	assert.Equal(t, "docs/dd", TitleFromFilename("docs/dd.md"))
}

func TestDocumentName(t *testing.T) {
	assert.Equal(t, "onboarding.md", DocumentName("docs/guides/onboarding.md"))
	assert.Equal(t, "onboarding.md", DocumentName("onboarding.md"))
}

func TestWriteGitHubOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "output")
	err := WriteGitHubOutput(path, &Result{Uploaded: 1, Replaced: 2, Skipped: 3, Failed: 0})
	require.NoError(t, err)
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "uploaded=1\nreplaced=2\nskipped=3\nfailed=0\n", string(body))
}
