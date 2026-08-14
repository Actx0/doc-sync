// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package core

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/Actx0/Gctx0"
	"github.com/samber/lo"
)

// Knowledge is the Actx0 documents API used during sync.
type Knowledge interface {
	List(ctx context.Context, limit, offset int) (*gctx0.DocumentList, error)
	Upload(ctx context.Context, file any, title string, labels map[string]string) (*gctx0.Document, error)
	Delete(ctx context.Context, documentID string) error
}

type Action string

const (
	ActionSkip    Action = "skip"
	ActionUpload  Action = "upload"
	ActionReplace Action = "replace"
	ActionError   Action = "error"
)

type FileResult struct {
	Path     string
	Filename string
	Action   Action
	ID       string
	Checksum string
	Err      error
}

type Result struct {
	Uploaded int
	Replaced int
	Skipped  int
	Failed   int
	Files    []FileResult
}

func NewKnowledge(cfg Config) *gctx0.Documents {
	return gctx0.NewKnowledge(
		gctx0.WithAccessKey(cfg.AccessKey),
		gctx0.WithWorkspaceId(cfg.WorkspaceID),
		gctx0.WithBaseURL(cfg.baseURL()),
		gctx0.WithTimeout(cfg.timeout()),
	)
}

func Run(ctx context.Context, cfg Config, out io.Writer) (*Result, error) {
	client := NewKnowledge(cfg)
	defer client.Close()
	return Sync(ctx, client, cfg, out)
}

func labelsEqual(docLabels []string, expected map[string]string) bool {
	want := lo.MapToSlice(expected, func(key, value string) string { return key + "=" + value })
	return len(docLabels) == len(want) && lo.Every(docLabels, want)
}

func listAll(ctx context.Context, client Knowledge) ([]gctx0.Document, error) {
	const pageSize = 100
	offset := 0
	var all []gctx0.Document
	for {
		listed, err := client.List(ctx, pageSize, offset)
		if err != nil {
			return nil, err
		}
		all = append(all, listed.Documents...)
		if listed.Total > 0 && offset+pageSize >= listed.Total {
			break
		}
		if len(listed.Documents) == 0 {
			break
		}
		offset += pageSize
	}
	return all, nil
}

func indexRemote(docs []gctx0.Document, labels map[string]string) map[string][]gctx0.Document {
	matched := lo.Filter(docs, func(doc gctx0.Document, _ int) bool {
		return labelsEqual(doc.Labels, labels)
	})
	return lo.GroupBy(matched, func(doc gctx0.Document) string { return doc.Filename })
}

func encodeLabels(labels map[string]string) []string {
	keys := lo.Keys(labels)
	slices.Sort(keys)
	return lo.Map(keys, func(key string, _ int) string { return key + "=" + labels[key] })
}

func deleteAll(ctx context.Context, client Knowledge, docs []gctx0.Document, dryRun bool) error {
	if dryRun {
		return nil
	}
	for _, doc := range docs {
		if err := client.Delete(ctx, doc.ID); err != nil {
			return fmt.Errorf("delete %s: %w", doc.ID, err)
		}
	}
	return nil
}

// Sync uploads local docs that are missing or whose checksum changed.
func Sync(ctx context.Context, client Knowledge, cfg Config, out io.Writer) (*Result, error) {
	if out == nil {
		out = io.Discard
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	labels, err := ParseLabels(cfg.Tags)
	if err != nil {
		return nil, err
	}

	repoDir, err := filepath.Abs(cfg.RepoDir)
	if err != nil {
		return nil, err
	}

	files, err := CollectFiles(repoDir, cfg.Paths)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no documents found")
	}

	remote, err := listAll(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("list knowledge: %w", err)
	}
	byName := indexRemote(remote, labels)

	result := &Result{Files: make([]FileResult, 0, len(files))}
	for _, absPath := range files {
		item := syncFile(ctx, client, repoDir, absPath, labels, byName, cfg.DryRun)
		result.Files = append(result.Files, item)
		switch item.Action {
		case ActionSkip:
			result.Skipped++
		case ActionUpload:
			result.Uploaded++
		case ActionReplace:
			result.Replaced++
		case ActionError:
			result.Failed++
		}
		fmt.Fprintln(out, formatFileResult(item))
		if item.Action != ActionError && item.Filename != "" {
			byName[item.Filename] = []gctx0.Document{{
				ID:       item.ID,
				Filename: item.Filename,
				Checksum: item.Checksum,
				Labels:   encodeLabels(labels),
			}}
		}
	}

	fmt.Fprintf(out, "\nuploaded=%d replaced=%d skipped=%d failed=%d\n",
		result.Uploaded, result.Replaced, result.Skipped, result.Failed)
	return result, nil
}

func syncFile(
	ctx context.Context,
	client Knowledge,
	repoDir, absPath string,
	labels map[string]string,
	byName map[string][]gctx0.Document,
	dryRun bool,
) FileResult {
	filename, err := RelFilename(repoDir, absPath)
	if err != nil {
		return FileResult{Path: absPath, Action: ActionError, Err: err}
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return FileResult{Path: absPath, Filename: filename, Action: ActionError, Err: err}
	}

	checksum := gctx0.FileChecksum(content)
	item := FileResult{Path: absPath, Filename: filename, Checksum: checksum}

	matching, extra := lo.FilterReject(byName[filename], func(doc gctx0.Document, _ int) bool {
		return doc.Checksum == checksum
	})
	if len(matching) > 0 {
		item.ID = matching[0].ID
		item.Action = ActionSkip
		if err := deleteAll(ctx, client, append(extra, matching[1:]...), dryRun); err != nil {
			item.Action = ActionError
			item.Err = err
		}
		return item
	}

	if dryRun {
		item.Action = lo.Ternary(len(extra) > 0, ActionReplace, ActionUpload)
		if len(extra) > 0 {
			item.ID = extra[0].ID
		}
		return item
	}

	if err := deleteAll(ctx, client, extra, false); err != nil {
		item.Action = ActionError
		item.Err = err
		return item
	}

	uploaded, err := client.Upload(ctx, gctx0.PreparedFile{
		Filename:    filename,
		Content:     content,
		ContentType: ContentType(filename),
	}, TitleFromFilename(filename), labels)
	if err != nil {
		item.Action = ActionError
		item.Err = err
		return item
	}

	item.ID = uploaded.ID
	if uploaded.Checksum != "" {
		item.Checksum = uploaded.Checksum
	}
	item.Action = lo.Ternary(len(extra) > 0, ActionReplace, ActionUpload)
	return item
}

func formatFileResult(item FileResult) string {
	switch item.Action {
	case ActionSkip:
		return fmt.Sprintf("skip    %s checksum=%s id=%s", item.Filename, item.Checksum, item.ID)
	case ActionUpload:
		return fmt.Sprintf("upload  %s -> %s checksum=%s", item.Filename, item.ID, item.Checksum)
	case ActionReplace:
		return fmt.Sprintf("replace %s -> %s checksum=%s", item.Filename, item.ID, item.Checksum)
	default:
		err := item.Err
		if err == nil {
			err = fmt.Errorf("unknown error")
		}
		name := item.Filename
		if name == "" {
			name = item.Path
		}
		return fmt.Sprintf("error   %s: %v", name, err)
	}
}
