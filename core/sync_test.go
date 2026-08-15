// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package core

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Actx0/Gctx0"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeKnowledge struct {
	docs    []gctx0.Document
	uploads int
	deletes []string
}

func (f *fakeKnowledge) List(context.Context, int, int) (*gctx0.DocumentList, error) {
	return &gctx0.DocumentList{Documents: f.docs, Total: len(f.docs)}, nil
}

func (f *fakeKnowledge) Upload(_ context.Context, file any, title string, labels map[string]string) (*gctx0.Document, error) {
	prepared, err := gctx0.PrepareFile(file)
	if err != nil {
		return nil, err
	}
	f.uploads++
	doc := gctx0.Document{
		ID:       "doc_new",
		Title:    title,
		Filename: prepared.Filename,
		Checksum: gctx0.FileChecksum(prepared.Content),
		Labels:   encodeLabels(labels),
	}
	f.docs = append(f.docs, doc)
	return &doc, nil
}

func (f *fakeKnowledge) Delete(_ context.Context, documentID string) error {
	f.deletes = append(f.deletes, documentID)
	f.docs = lo.Filter(f.docs, func(doc gctx0.Document, _ int) bool { return doc.ID != documentID })
	return nil
}

func TestSyncChecksums(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	require.NoError(t, os.MkdirAll(docs, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(docs, "dd.md"), []byte("dd"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(docs, "ree.md"), []byte("ree"), 0o644))

	labels := map[string]string{"tag": "docs"}
	same := gctx0.FileChecksum([]byte("dd"))
	stale := gctx0.FileChecksum([]byte("old"))
	client := &fakeKnowledge{docs: []gctx0.Document{
		{ID: "doc_dd", Filename: "dd.md", Checksum: same, Labels: []string{"tag=docs"}},
		{ID: "doc_old", Filename: "ree.md", Checksum: stale, Labels: []string{"tag=docs"}},
		{ID: "doc_other", Filename: "dd.md", Checksum: same, Labels: []string{"team=other"}},
	}}

	var buf bytes.Buffer
	result, err := Sync(context.Background(), client, Config{
		WorkspaceID: "ws",
		AccessKey:   "key",
		Tags:        "tag: docs",
		Paths:       []string{"docs/dd.md", "docs/ree.md"},
		RepoDir:     root,
	}, &buf)
	require.NoError(t, err)
	require.Len(t, result.Files, 2)
	assert.Equal(t, 1, result.Skipped)
	assert.Equal(t, 1, result.Replaced)
	assert.Equal(t, 0, result.Uploaded)
	assert.Equal(t, []string{"doc_old"}, client.deletes)
	assert.Equal(t, 1, client.uploads)
	assert.Equal(t, ActionSkip, result.Files[0].Action)
	assert.Equal(t, "doc_dd", result.Files[0].ID)
	assert.Equal(t, ActionReplace, result.Files[1].Action)
	assert.Contains(t, buf.String(), "skip    docs/dd.md")
	assert.Contains(t, buf.String(), "replace docs/ree.md")
	assert.True(t, labelsEqual([]string{"tag=docs"}, labels))
}

func TestSyncSkipsBasenameRemote(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "docs", "guides"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "docs", "guides", "onboarding.md"), []byte("hello"), 0o644))

	same := gctx0.FileChecksum([]byte("hello"))
	client := &fakeKnowledge{docs: []gctx0.Document{
		{ID: "doc_1", Filename: "onboarding.md", Checksum: same, Labels: []string{"tag=docs"}},
		{ID: "doc_2", Filename: "onboarding.md", Checksum: same, Labels: []string{"tag=docs"}},
	}}

	result, err := Sync(context.Background(), client, Config{
		WorkspaceID: "ws",
		AccessKey:   "key",
		Tags:        "tag: docs",
		Paths:       []string{"docs/guides/onboarding.md"},
		RepoDir:     root,
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Skipped)
	assert.Equal(t, 0, result.Uploaded)
	assert.Equal(t, 0, client.uploads)
	assert.Equal(t, []string{"doc_2"}, client.deletes)
	assert.Equal(t, "doc_1", result.Files[0].ID)
}

func TestSyncDuplicateBasenames(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "docs", "a"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "docs", "b"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "docs", "a", "notes.md"), []byte("a"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "docs", "b", "notes.md"), []byte("b"), 0o644))

	_, err := Sync(context.Background(), &fakeKnowledge{}, Config{
		WorkspaceID: "ws",
		AccessKey:   "key",
		Paths:       []string{"docs/"},
		RepoDir:     root,
	}, nil)
	require.ErrorContains(t, err, "duplicate filenames: notes.md")
}

func TestSyncDryRun(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "docs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "docs", "dd.md"), []byte("dd"), 0o644))

	client := &fakeKnowledge{}
	result, err := Sync(context.Background(), client, Config{
		WorkspaceID: "ws",
		AccessKey:   "key",
		Paths:       []string{"docs/dd.md"},
		RepoDir:     root,
		DryRun:      true,
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Uploaded)
	assert.Equal(t, 0, client.uploads)
	assert.Empty(t, client.deletes)
}

func TestConfigValidate(t *testing.T) {
	err := Config{WorkspaceID: "ws", AccessKey: "key", Paths: []string{"docs/"}, RepoDir: t.TempDir()}.Validate()
	require.NoError(t, err)

	err = Config{AccessKey: "key", Paths: []string{"docs/"}, RepoDir: t.TempDir()}.Validate()
	require.EqualError(t, err, "workspace id is required")
}
