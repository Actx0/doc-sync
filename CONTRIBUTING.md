# Contributing to doc-sync

Thanks for your interest in contributing to the Actx0 docs sync GitHub Action.

## Development setup

```bash
git clone https://github.com/Actx0/doc-sync.git
cd doc-sync
go test ./...
go build -o doc-sync ./cmd
goreleaser check
```

Push a `v*` tag to publish binaries with GoReleaser.

## Pull requests

1. Open an issue first for larger changes so we can align on scope.
2. Keep changes focused and consistent with existing package style.
3. Add or update tests for behavior you change.
4. Run `go test ./...` before opening the PR.
5. Use a clear, lowercase conventional commit subject when possible
   (for example `feat: skip unchanged docs by checksum`).

## Code style

- Prefer small, readable helpers over deep abstractions.
- Match existing naming and blank-line spacing in nearby files.
- Do not commit secrets, access keys, or local env files.

## Security issues

Do not open a public issue for security problems. See [SECURITY.md](SECURITY.md).
