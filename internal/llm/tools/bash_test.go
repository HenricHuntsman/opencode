package tools

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/permission"
	"github.com/opencode-ai/opencode/internal/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSafeReadOnly(t *testing.T) {
	safe := []string{
		"ls", "ls -la", "pwd", "git status", "git log --oneline", "git diff HEAD", "go version", "go vet ./...",
	}
	for _, c := range safe {
		assert.True(t, isSafeReadOnly(c), "expected safe: %q", c)
	}

	needsApproval := map[string][]string{
		"modifies files or runs programs": {
			"rm -rf x", "touch x", "go run main.go", "go install ./...", "go fmt ./...", "go mod tidy", "go test ./...",
			"git branch -D x", "git tag v1", "git remote add o u", "kill 1", "env sh", "nohup sh", "timeout 1 sh", "time sh",
		},
		"compound commands": {
			"ls; rm -rf x", "ls && rm x", "ls || rm x", "ls | sh", "ls & rm x", "ls\nrm x",
		},
		"substitution and expansion": {
			"ls $(rm x)", "ls `rm x`", "echo $HOME", "ls \"a\"", "ls 'a'", "ls a\\ b",
		},
		"redirection": {
			"ls > out", "ls >> out", "echo hi > f", "cat < f",
		},
		"output-writing flags": {
			"git diff --output=x", "git log --output x", "git diff --ext-diff",
		},
		"prefix lookalikes": {
			"lsof", "lsblk", "pwdx", "dateutils", "git statusx", "git log-x", "go versionx",
		},
		"empty": {"", "   ", "\t"},
	}
	for group, cmds := range needsApproval {
		for _, c := range cmds {
			assert.False(t, isSafeReadOnly(c), "%s: expected approval for %q", group, c)
		}
	}
}

func TestBannedCommandIn(t *testing.T) {
	banned := []string{
		"curl example.com", "wget x", "ls; curl x", "ls && wget x", "echo hi | nc host 1", "$(curl x)",
		"`curl x`", "sudo curl x", "env FOO=1 curl x", "nohup wget x", "/usr/bin/curl x", "FOO=bar curl x", "CURL x",
	}
	for _, c := range banned {
		_, ok := bannedCommandIn(c)
		assert.True(t, ok, "expected banned: %q", c)
	}
	for _, c := range []string{"ls", "echo curl", "go test ./...", "git log"} {
		_, ok := bannedCommandIn(c)
		assert.False(t, ok, "expected allowed: %q", c)
	}
}

type fakePermissions struct {
	*pubsub.Broker[permission.PermissionRequest]
	allow    bool
	requests []permission.CreatePermissionRequest
}

func newFakePermissions(allow bool) *fakePermissions {
	return &fakePermissions{Broker: pubsub.NewBroker[permission.PermissionRequest](), allow: allow}
}

func (f *fakePermissions) GrantPersistant(permission.PermissionRequest) {}
func (f *fakePermissions) Grant(permission.PermissionRequest)           {}
func (f *fakePermissions) Deny(permission.PermissionRequest)            {}
func (f *fakePermissions) AutoApproveSession(string)                    {}
func (f *fakePermissions) Request(o permission.CreatePermissionRequest) bool {
	f.requests = append(f.requests, o)
	return f.allow
}

func runBash(t *testing.T, perms permission.Service, command string) (ToolResponse, error) {
	t.Helper()
	input, err := json.Marshal(BashParams{Command: command})
	require.NoError(t, err)
	ctx := context.WithValue(context.Background(), SessionIDContextKey, "session")
	ctx = context.WithValue(ctx, MessageIDContextKey, "message")
	return NewBashTool(perms).Run(ctx, ToolCall{Name: BashToolName, Input: string(input)})
}

func TestBashTool_Permissions(t *testing.T) {
	dir := t.TempDir()
	_, err := config.Load(dir, false)
	require.NoError(t, err)
	config.Get().WorkingDir = dir

	t.Run("harmless command runs without a prompt", func(t *testing.T) {
		perms := newFakePermissions(false)
		resp, err := runBash(t, perms, "pwd")
		require.NoError(t, err)
		assert.False(t, resp.IsError)
		assert.Empty(t, perms.requests)
	})

	t.Run("approved file-modifying command runs after a prompt", func(t *testing.T) {
		perms := newFakePermissions(true)
		target := filepath.Join(dir, "approved.txt")
		_, err := runBash(t, perms, "touch "+target)
		require.NoError(t, err)
		require.Len(t, perms.requests, 1)
		assert.Equal(t, BashToolName, perms.requests[0].ToolName)
		_, statErr := os.Stat(target)
		assert.NoError(t, statErr)
	})

	t.Run("denied command does not run", func(t *testing.T) {
		perms := newFakePermissions(false)
		target := filepath.Join(dir, "denied.txt")
		_, err := runBash(t, perms, "touch "+target)
		assert.True(t, errors.Is(err, permission.ErrorPermissionDenied))
		require.Len(t, perms.requests, 1)
		_, statErr := os.Stat(target)
		assert.True(t, os.IsNotExist(statErr), "denied command must not execute")
	})

	t.Run("compound command hiding a write is prompted and not run when denied", func(t *testing.T) {
		perms := newFakePermissions(false)
		target := filepath.Join(dir, "chained.txt")
		_, err := runBash(t, perms, "ls; touch "+target)
		assert.True(t, errors.Is(err, permission.ErrorPermissionDenied))
		require.Len(t, perms.requests, 1)
		_, statErr := os.Stat(target)
		assert.True(t, os.IsNotExist(statErr))
	})

	t.Run("banned command is rejected before any prompt", func(t *testing.T) {
		perms := newFakePermissions(true)
		resp, err := runBash(t, perms, "ls; curl example.com")
		require.NoError(t, err)
		assert.True(t, resp.IsError)
		assert.Contains(t, resp.Content, "not allowed")
		assert.Empty(t, perms.requests)
	})

	t.Run("empty and whitespace-only commands are rejected cleanly", func(t *testing.T) {
		for _, c := range []string{"", "   ", "\t\n"} {
			perms := newFakePermissions(true)
			resp, err := runBash(t, perms, c)
			require.NoError(t, err)
			assert.True(t, resp.IsError)
			assert.Contains(t, resp.Content, "missing command")
			assert.Empty(t, perms.requests)
		}
	})
}
