package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/diff"
	"github.com/opencode-ai/opencode/internal/history"
	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/opencode-ai/opencode/internal/lsp"
	"github.com/opencode-ai/opencode/internal/permission"
)

type PatchParams struct {
	PatchText string `json:"patch_text"`
}

type PatchResponseMetadata struct {
	FilesChanged []string `json:"files_changed"`
	Additions    int      `json:"additions"`
	Removals     int      `json:"removals"`
}

type patchTool struct {
	lspClients  map[string]*lsp.Client
	permissions permission.Service
	files       history.Service
}

const (
	PatchToolName    = "patch"
	patchDescription = `Applies a patch to multiple files in one operation. This tool is useful for making coordinated changes across multiple files.

The patch text must follow this format:
*** Begin Patch
*** Update File: /path/to/file
@@ Context line (unique within the file)
 Line to keep
-Line to remove
+Line to add
 Line to keep
*** Add File: /path/to/new/file
+Content of the new file
+More content
*** Delete File: /path/to/file/to/delete
*** End Patch

Before using this tool:
1. Use the FileRead tool to understand the files' contents and context
2. Verify all file paths are correct (use the LS tool)

CRITICAL REQUIREMENTS FOR USING THIS TOOL:

1. UNIQUENESS: Context lines MUST uniquely identify the specific sections you want to change
2. PRECISION: All whitespace, indentation, and surrounding code must match exactly
3. VALIDATION: Ensure edits result in idiomatic, correct code
4. PATHS: Always use absolute file paths (starting with /)

All file previews are reviewed before the patch is applied.`
)

func NewPatchTool(lspClients map[string]*lsp.Client, permissions permission.Service, files history.Service) BaseTool {
	return &patchTool{
		lspClients:  lspClients,
		permissions: permissions,
		files:       files,
	}
}

func (p *patchTool) Info() ToolInfo {
	return ToolInfo{
		Name:        PatchToolName,
		Description: patchDescription,
		Parameters: map[string]any{
			"patch_text": map[string]any{
				"type":        "string",
				"description": "The full patch text that describes all changes to be made",
			},
		},
		Required: []string{"patch_text"},
	}
}

func (p *patchTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params PatchParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("invalid parameters"), nil
	}

	if params.PatchText == "" {
		return NewTextErrorResponse("patch_text is required"), nil
	}

	// Identify all files needed for the patch and verify they've been read
	filesToRead := diff.IdentifyFilesNeeded(params.PatchText)
	for _, filePath := range filesToRead {
		absPath := filePath
		if !filepath.IsAbs(absPath) {
			wd := config.WorkingDirectory()
			absPath = filepath.Join(wd, absPath)
		}

		if getLastReadTime(absPath).IsZero() {
			return NewTextErrorResponse(fmt.Sprintf("you must read the file %s before patching it. Use the FileRead tool first", filePath)), nil
		}

		fileInfo, err := os.Stat(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				return NewTextErrorResponse(fmt.Sprintf("file not found: %s", absPath)), nil
			}
			return ToolResponse{}, fmt.Errorf("failed to access file: %w", err)
		}

		if fileInfo.IsDir() {
			return NewTextErrorResponse(fmt.Sprintf("path is a directory, not a file: %s", absPath)), nil
		}

		modTime := fileInfo.ModTime()
		lastRead := getLastReadTime(absPath)
		if modTime.After(lastRead) {
			return NewTextErrorResponse(
				fmt.Sprintf("file %s has been modified since it was last read (mod time: %s, last read: %s)",
					absPath, modTime.Format(time.RFC3339), lastRead.Format(time.RFC3339),
				)), nil
		}
	}

	// Check for new files to ensure they don't already exist
	filesToAdd := diff.IdentifyFilesAdded(params.PatchText)
	for _, filePath := range filesToAdd {
		absPath := filePath
		if !filepath.IsAbs(absPath) {
			wd := config.WorkingDirectory()
			absPath = filepath.Join(wd, absPath)
		}

		_, err := os.Stat(absPath)
		if err == nil {
			return NewTextErrorResponse(fmt.Sprintf("file already exists and cannot be added: %s", absPath)), nil
		} else if !os.IsNotExist(err) {
			return ToolResponse{}, fmt.Errorf("failed to check file: %w", err)
		}
	}

	// Load all required files
	currentFiles := make(map[string]string)
	for _, filePath := range filesToRead {
		absPath := filePath
		if !filepath.IsAbs(absPath) {
			wd := config.WorkingDirectory()
			absPath = filepath.Join(wd, absPath)
		}

		content, err := os.ReadFile(absPath)
		if err != nil {
			return ToolResponse{}, fmt.Errorf("failed to read file %s: %w", absPath, err)
		}
		currentFiles[filePath] = string(content)
	}

	// Process the patch
	patch, fuzz, err := diff.TextToPatch(params.PatchText, currentFiles)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("failed to parse patch: %s", err)), nil
	}

	if fuzz > 3 {
		return NewTextErrorResponse(fmt.Sprintf("patch contains fuzzy matches (fuzz level: %d). Please make your context lines more precise", fuzz)), nil
	}

	// Convert patch to commit
	commit, err := diff.PatchToCommit(patch, currentFiles)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("failed to create commit from patch: %s", err)), nil
	}

	// Get session ID and message ID
	sessionID, messageID := GetContextValues(ctx)
	if sessionID == "" || messageID == "" {
		return ToolResponse{}, fmt.Errorf("session ID and message ID are required for creating a patch")
	}

	// Keep review order stable and reject overlapping source and destination paths.
	paths := make([]string, 0, len(commit.Changes))
	targets := make(map[string]struct{})
	newFiles := make(map[string]bool)
	for path, change := range commit.Changes {
		paths = append(paths, path)
		if err := reservePatchPath(targets, path); err != nil {
			return NewTextErrorResponse(err.Error()), nil
		}
		if change.Type == diff.ActionAdd {
			newFiles[patchAbsolutePath(path)] = true
		}
		if change.MovePath != nil {
			newFiles[patchAbsolutePath(*change.MovePath)] = true
			if err := reservePatchPath(targets, *change.MovePath); err != nil {
				return NewTextErrorResponse(err.Error()), nil
			}
		}
	}
	sort.Strings(paths)
	if err := checkPatchFiles(commit, paths); err != nil {
		return NewTextErrorResponse(err.Error()), nil
	}

	// Request permission for all changes before applying any.
	for _, path := range paths {
		change := commit.Changes[path]
		switch change.Type {
		case diff.ActionAdd:
			if !p.reviewChange(sessionID, path, "create", "", *change.NewContent) {
				return ToolResponse{}, permission.ErrorPermissionDenied
			}
		case diff.ActionDelete:
			if !p.reviewChange(sessionID, path, "delete", *change.OldContent, "") {
				return ToolResponse{}, permission.ErrorPermissionDenied
			}
		case diff.ActionUpdate:
			if change.MovePath != nil {
				if !p.reviewChange(sessionID, path, "delete", *change.OldContent, "") ||
					!p.reviewChange(sessionID, *change.MovePath, "create", "", *change.NewContent) {
					return ToolResponse{}, permission.ErrorPermissionDenied
				}
			} else if !p.reviewChange(sessionID, path, "update", *change.OldContent, *change.NewContent) {
				return ToolResponse{}, permission.ErrorPermissionDenied
			}
		}
	}

	// A user or another process may edit a file while approval is open.
	if err := ctx.Err(); err != nil {
		return ToolResponse{}, err
	}
	if err := checkPatchFiles(commit, paths); err != nil {
		return NewTextErrorResponse(err.Error()), nil
	}

	// Apply the changes to the filesystem.
	err = diff.ApplyCommit(commit, func(path string, content string) error {
		absPath := path
		if !filepath.IsAbs(absPath) {
			wd := config.WorkingDirectory()
			absPath = filepath.Join(wd, absPath)
		}

		// Create parent directories if needed
		dir := filepath.Dir(absPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create parent directories for %s: %w", absPath, err)
		}

		if newFiles[filepath.Clean(absPath)] {
			file, err := os.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
			if err != nil {
				return err
			}
			_, writeErr := file.WriteString(content)
			closeErr := file.Close()
			if writeErr != nil {
				return writeErr
			}
			return closeErr
		}
		return os.WriteFile(absPath, []byte(content), 0o644)
	}, func(path string) error {
		absPath := path
		if !filepath.IsAbs(absPath) {
			wd := config.WorkingDirectory()
			absPath = filepath.Join(wd, absPath)
		}
		return os.Remove(absPath)
	})
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("failed to apply patch: %s", err)), nil
	}

	// A move changes both paths. Keep both in history, diagnostics, and the tool result.
	for _, path := range paths {
		change := commit.Changes[path]
		if change.MovePath != nil {
			commit.Changes[*change.MovePath] = diff.FileChange{Type: diff.ActionAdd, NewContent: change.NewContent}
			commit.Changes[path] = diff.FileChange{Type: diff.ActionDelete, OldContent: change.OldContent}
			paths = append(paths, *change.MovePath)
		}
	}
	sort.Strings(paths)

	// Update file history for all modified files.
	changedFiles := []string{}
	totalAdditions := 0
	totalRemovals := 0

	for _, path := range paths {
		change := commit.Changes[path]
		absPath := path
		if !filepath.IsAbs(absPath) {
			wd := config.WorkingDirectory()
			absPath = filepath.Join(wd, absPath)
		}
		changedFiles = append(changedFiles, absPath)

		oldContent := ""
		if change.OldContent != nil {
			oldContent = *change.OldContent
		}

		newContent := ""
		if change.NewContent != nil {
			newContent = *change.NewContent
		}

		// Calculate diff statistics
		_, additions, removals := diff.GenerateDiff(oldContent, newContent, path)
		totalAdditions += additions
		totalRemovals += removals

		// Update history
		file, err := p.files.GetByPathAndSession(ctx, absPath, sessionID)
		if err != nil && change.Type != diff.ActionAdd {
			// If not adding a file, create history entry for existing file
			_, err = p.files.Create(ctx, sessionID, absPath, oldContent)
			if err != nil {
				logging.Debug("Error creating file history", "error", err)
			}
		}

		if err == nil && change.Type != diff.ActionAdd && file.Content != oldContent {
			// User manually changed content, store intermediate version
			_, err = p.files.CreateVersion(ctx, sessionID, absPath, oldContent)
			if err != nil {
				logging.Debug("Error creating file history version", "error", err)
			}
		}

		// Store new version
		if change.Type == diff.ActionDelete {
			_, err = p.files.CreateVersion(ctx, sessionID, absPath, "")
		} else {
			_, err = p.files.CreateVersion(ctx, sessionID, absPath, newContent)
		}
		if err != nil {
			logging.Debug("Error creating file history version", "error", err)
		}

		// Record file operations
		recordFileWrite(absPath)
		recordFileRead(absPath)
	}

	// Run LSP diagnostics on all changed files
	for _, filePath := range changedFiles {
		waitForLspDiagnostics(ctx, filePath, p.lspClients)
	}

	result := fmt.Sprintf("Patch applied successfully. %d files changed, %d additions, %d removals",
		len(changedFiles), totalAdditions, totalRemovals)

	diagnosticsText := ""
	for _, filePath := range changedFiles {
		diagnosticsText += getDiagnostics(filePath, p.lspClients)
	}

	if diagnosticsText != "" {
		result += "\n\nDiagnostics:\n" + diagnosticsText
	}

	return WithResponseMetadata(
		NewTextResponse(result),
		PatchResponseMetadata{
			FilesChanged: changedFiles,
			Additions:    totalAdditions,
			Removals:     totalRemovals,
		}), nil
}

func (p *patchTool) reviewChange(sessionID, path, action, before, after string) bool {
	path = patchAbsolutePath(path)
	preview, _, _ := diff.GenerateDiff(before, after, path)
	return p.permissions.Request(permission.CreatePermissionRequest{
		SessionID:   sessionID,
		Path:        path,
		ToolName:    PatchToolName,
		Action:      action,
		Description: fmt.Sprintf("Patch: %s %s", action, path),
		Params:      EditPermissionsParams{FilePath: path, Diff: preview},
	})
}

func patchAbsolutePath(path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(config.WorkingDirectory(), path)
	}
	return filepath.Clean(path)
}

func reservePatchPath(targets map[string]struct{}, path string) error {
	path = patchAbsolutePath(path)
	if _, exists := targets[path]; exists {
		return fmt.Errorf("patch uses file %s more than once", path)
	}
	targets[path] = struct{}{}
	return nil
}

func checkPatchFiles(commit diff.Commit, paths []string) error {
	for _, path := range paths {
		change := commit.Changes[path]
		absolute := patchAbsolutePath(path)
		if change.OldContent == nil {
			if err := checkPatchDestination(absolute); err != nil {
				return err
			}
		} else {
			current, err := os.ReadFile(absolute)
			if err != nil {
				return fmt.Errorf("cannot verify reviewed file %s: %w", path, err)
			}
			if string(current) != *change.OldContent {
				return fmt.Errorf("file %s changed after the patch was prepared; read it again and request a new review", path)
			}
		}
		if change.MovePath != nil {
			if err := checkPatchDestination(patchAbsolutePath(*change.MovePath)); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkPatchDestination(path string) error {
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		if err != nil {
			return fmt.Errorf("cannot verify patch destination %s: %w", path, err)
		}
		return fmt.Errorf("patch destination %s already exists; prepare a new patch", path)
	}
	return nil
}
