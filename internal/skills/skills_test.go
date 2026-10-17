package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validSkill = `---
name: git-release
description: Create consistent releases
license: MIT
metadata:
  opencode/autoinvoke: "false"
---

## What I do

Draft notes.
`

func TestParse(t *testing.T) {
	t.Run("valid skill", func(t *testing.T) {
		s, err := Parse([]byte(validSkill), "git-release")
		require.NoError(t, err)
		assert.Equal(t, "git-release", s.Name)
		assert.Equal(t, "Create consistent releases", s.Description)
		assert.Equal(t, "MIT", s.License)
		assert.False(t, s.AutoInvoke())
		assert.Equal(t, "## What I do\n\nDraft notes.", s.Body)
	})

	t.Run("accepts yaml boolean metadata", func(t *testing.T) {
		doc := "---\nname: a\ndescription: d\nmetadata:\n  opencode/autoinvoke: false\n---\nbody"
		s, err := Parse([]byte(doc), "a")
		require.NoError(t, err)
		assert.False(t, s.AutoInvoke())
	})

	t.Run("autoinvoke defaults to true", func(t *testing.T) {
		s, err := Parse([]byte("---\nname: a\ndescription: d\n---\nbody"), "a")
		require.NoError(t, err)
		assert.True(t, s.AutoInvoke())
	})

	t.Run("handles CRLF line endings", func(t *testing.T) {
		_, err := Parse([]byte("---\r\nname: a\r\ndescription: d\r\n---\r\nbody"), "a")
		require.NoError(t, err)
	})

	invalid := map[string]string{
		"no frontmatter":      "just text",
		"unterminated":        "---\nname: a\ndescription: d\n",
		"missing name":        "---\ndescription: d\n---\n",
		"missing description": "---\nname: a\n---\n",
		"uppercase name":      "---\nname: Abc\ndescription: d\n---\n",
		"consecutive hyphens": "---\nname: a--b\ndescription: d\n---\n",
		"leading hyphen":      "---\nname: -a\ndescription: d\n---\n",
		"trailing hyphen":     "---\nname: a-\ndescription: d\n---\n",
		"directory mismatch":  "---\nname: other\ndescription: d\n---\n",
		"malformed yaml":      "---\nname: [a\ndescription: d\n---\n",
	}
	for name, doc := range invalid {
		t.Run("rejects "+name, func(t *testing.T) {
			_, err := Parse([]byte(doc), "a")
			assert.Error(t, err)
		})
	}

	t.Run("rejects overlong name and description", func(t *testing.T) {
		long := make([]byte, 65)
		for i := range long {
			long[i] = 'a'
		}
		_, err := Parse([]byte("---\nname: "+string(long)+"\ndescription: d\n---\n"), string(long))
		assert.Error(t, err)

		longDesc := make([]byte, 1025)
		for i := range longDesc {
			longDesc[i] = 'x'
		}
		_, err = Parse([]byte("---\nname: a\ndescription: "+string(longDesc)+"\n---\n"), "a")
		assert.Error(t, err)
	})
}

func writeSkill(t *testing.T, root, name, desc string) {
	t.Helper()
	dir := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	doc := "---\nname: " + name + "\ndescription: " + desc + "\n---\nbody\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(doc), 0o644))
}

func TestDiscover(t *testing.T) {
	t.Run("walks up to git root and prefers project over global", func(t *testing.T) {
		base := t.TempDir()
		repo := filepath.Join(base, "repo")
		sub := filepath.Join(repo, "pkg", "deep")
		home := filepath.Join(base, "home")
		require.NoError(t, os.MkdirAll(filepath.Join(repo, ".git"), 0o755))
		require.NoError(t, os.MkdirAll(sub, 0o755))

		writeSkill(t, filepath.Join(repo, ".opencode", "skills"), "shared", "project version")
		writeSkill(t, filepath.Join(repo, ".claude", "skills"), "claude-only", "claude compat")
		writeSkill(t, filepath.Join(repo, "pkg", ".agents", "skills"), "agents-only", "agents compat")
		writeSkill(t, filepath.Join(home, ".config", "opencode", "skills"), "shared", "global version")
		writeSkill(t, filepath.Join(home, ".config", "opencode", "skills"), "global-only", "global")
		// Above the git root: must not be discovered.
		writeSkill(t, filepath.Join(base, ".opencode", "skills"), "outside", "outside repo")

		got, warnings := Discover(sub, home)
		assert.Empty(t, warnings)

		byName := map[string]Skill{}
		for _, s := range got {
			byName[s.Name] = s
		}
		assert.Equal(t, "project version", byName["shared"].Description)
		assert.Contains(t, byName, "claude-only")
		assert.Contains(t, byName, "agents-only")
		assert.Contains(t, byName, "global-only")
		assert.NotContains(t, byName, "outside")
		assert.Len(t, got, 4)
	})

	t.Run("skips invalid skills and reports warnings", func(t *testing.T) {
		repo := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(repo, ".git"), 0o755))
		root := filepath.Join(repo, ".opencode", "skills")
		writeSkill(t, root, "good", "ok")
		bad := filepath.Join(root, "bad")
		require.NoError(t, os.MkdirAll(bad, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(bad, "SKILL.md"), []byte("no frontmatter"), 0o644))

		got, warnings := Discover(repo, "")
		require.Len(t, got, 1)
		assert.Equal(t, "good", got[0].Name)
		assert.Len(t, warnings, 1)
	})

	t.Run("returns nothing when no skills exist", func(t *testing.T) {
		repo := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(repo, ".git"), 0o755))
		got, warnings := Discover(repo, "")
		assert.Empty(t, got)
		assert.Empty(t, warnings)
	})
}
