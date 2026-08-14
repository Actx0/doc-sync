### Actx0 Doc Sync

GitHub Action that syncs repository docs into an Actx0 workspace knowledge base. Unchanged files are skipped by checksum; changed files replace the previous document with the same filename and labels.

#### GitHub Action

```yaml
- uses: Actx0/doc-sync@v1
  with:
    workspace_id: ${{ secrets.ACTX0_WORKSPACE_ID }}
    access_key: ${{ secrets.ACTX0_ACCESS_KEY }}
    tags: |
      tag: docs
      repo: ${{ github.repository }}
      team: platform
    paths: |
      docs/
      docs/*.md
      docs/file1.md
      docs/file2.txt
    repo_dir: ${{ github.workspace }}
```

- `workspace_id` (required) — Actx0 workspace ID
- `access_key` (required) — Actx0 access key
- `tags` — knowledge labels, one `key: value` per line
- `paths` (required) — files, directories, or globs to sync, one per line. Directories include markdown and text files
- `repo_dir` — repository root used to resolve paths. Defaults to `github.workspace`
- `base_url` — Actx0 API base URL. Defaults to `https://app.actx0.com`
- `dry_run` — compare checksums without uploading or deleting

Outputs: `uploaded`, `replaced`, `skipped`, `failed`.

Tagged versions (`@v1` or `@v1.0.0`) download the GoReleaser binary for the runner OS. `@main` builds from source.

#### CLI

```zsh
go install github.com/Actx0/doc-sync/cmd/doc-sync@latest

doc-sync \
  --workspace-id "$ACTX0_WORKSPACE_ID" \
  --access-key "$ACTX0_ACCESS_KEY" \
  --tags $'tag: docs\nteam: platform\nrepo: actx0/app' \
  --repo-dir . \
  --path docs/ \
  --path 'docs/*.md' \
  --path docs/file1.md \
  --path docs/file2.txt
```

Release binaries are published by GoReleaser when a `v*` tag is pushed.

#### Development

```zsh
go test ./...
go build -o doc-sync ./cmd/doc-sync
goreleaser check
```
