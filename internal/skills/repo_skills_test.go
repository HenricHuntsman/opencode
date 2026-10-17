package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Guards the skills shipped in this repository against invalid frontmatter.
func TestRepoSkillsAreValid(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)

	dirs, err := os.ReadDir(filepath.Join(root, ".opencode", "skills"))
	require.NoError(t, err)
	require.NotEmpty(t, dirs)

	found, warnings := Discover(root, "")
	assert.Empty(t, warnings, "repo skills must all be valid")

	names := map[string]bool{}
	for _, s := range found {
		names[s.Name] = true
	}
	for _, d := range dirs {
		if d.IsDir() {
			assert.True(t, names[d.Name()], "skill %q was not discovered", d.Name())
		}
	}
	assert.True(t, names["go-verify"])
	assert.True(t, names["github-release"])
}
