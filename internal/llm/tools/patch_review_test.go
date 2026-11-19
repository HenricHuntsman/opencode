package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/opencode-ai/opencode/internal/diff"
	"github.com/stretchr/testify/require"
)

func TestCheckPatchFilesRejectsChangesMadeDuringReview(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "original.txt")
	destination := filepath.Join(dir, "destination.txt")
	before := "original"
	require.NoError(t, os.WriteFile(source, []byte(before), 0o600))
	commit := diff.Commit{Changes: map[string]diff.FileChange{
		source: {Type: diff.ActionUpdate, OldContent: &before, MovePath: &destination},
	}}
	paths := []string{source}
	require.NoError(t, checkPatchFiles(commit, paths))

	require.NoError(t, os.WriteFile(source, []byte("user edit"), 0o600))
	require.ErrorContains(t, checkPatchFiles(commit, paths), "changed after the patch was prepared")
	require.NoError(t, os.WriteFile(source, []byte(before), 0o600))
	require.NoError(t, os.WriteFile(destination, []byte("user file"), 0o600))
	require.ErrorContains(t, checkPatchFiles(commit, paths), "already exists")
}
