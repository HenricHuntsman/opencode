package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/llm/models"
	"github.com/opencode-ai/opencode/internal/logging"
)

func GetAgentPrompt(agentName config.AgentName, provider models.ModelProvider) string {
	basePrompt := ""
	switch agentName {
	case config.AgentCoder:
		basePrompt = CoderPrompt(provider)
	case config.AgentTitle:
		basePrompt = TitlePrompt(provider)
	case config.AgentTask:
		basePrompt = TaskPrompt(provider)
	case config.AgentSummarizer:
		basePrompt = SummarizerPrompt(provider)
	default:
		basePrompt = "You are a helpful assistant"
	}

	if agentName == config.AgentCoder || agentName == config.AgentTask {
		// Add context from project-specific instruction files if they exist
		contextContent := getContextFromPaths()
		logging.Debug("Context content", "Context", contextContent)
		if contextContent != "" {
			return fmt.Sprintf("%s\n\n# Project-Specific Context\n Make sure to follow the instructions in the context below\n%s", basePrompt, contextContent)
		}
	}
	return basePrompt
}

func getContextFromPaths() string {
	var (
		cfg          = config.Get()
		workDir      = cfg.WorkingDir
		contextPaths = cfg.ContextPaths
	)

	return processContextPaths(workDir, contextPaths)
}

func processContextPaths(workDir string, paths []string) string {
	// Track processed files to avoid duplicates
	processedFiles := make(map[string]bool)

	results := make([]string, 0)
	for _, p := range paths {
		if strings.HasSuffix(p, "/") {
			filepath.WalkDir(filepath.Join(workDir, p), func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				lowerPath := strings.ToLower(path)
				if processedFiles[lowerPath] {
					return nil
				}
				processedFiles[lowerPath] = true

				if result := processFile(path); result != "" {
					results = append(results, result)
				}
				return nil
			})
		} else {
			fullPath := filepath.Join(workDir, p)

			lowerPath := strings.ToLower(fullPath)
			if processedFiles[lowerPath] {
				continue
			}
			processedFiles[lowerPath] = true

			if result := processFile(fullPath); result != "" {
				results = append(results, result)
			}
		}
	}

	return strings.Join(results, "\n")
}

func processFile(filePath string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	return "# From:" + filePath + "\n" + string(content)
}
