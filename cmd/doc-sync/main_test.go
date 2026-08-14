// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Actx0/doc-sync/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteGitHubOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "output")
	err := writeGitHubOutput(path, &core.Result{Uploaded: 1, Replaced: 2, Skipped: 3, Failed: 0})
	require.NoError(t, err)
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "uploaded=1\nreplaced=2\nskipped=3\nfailed=0\n", string(body))
}
