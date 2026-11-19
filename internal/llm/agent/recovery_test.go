package agent

import (
	"testing"

	"github.com/opencode-ai/opencode/internal/message"
	"github.com/stretchr/testify/require"
)

func TestRecoverToolHistoryKeepsSavedResultsAndMarksUnknownOutcomes(t *testing.T) {
	assistant := message.Message{Role: message.Assistant, SessionID: "session"}
	assistant.AddToolCall(message.ToolCall{ID: "saved", Name: "read", Input: "{}"})
	assistant.AddToolCall(message.ToolCall{ID: "interrupted", Name: "patch", Input: "{"})
	assistant.AddToolCall(message.ToolCall{Input: "{"})
	saved := message.Message{Role: message.Tool, SessionID: "session"}
	saved.AddToolResult(message.ToolResult{ToolCallID: "saved", Content: "file contents"})
	followUp := message.Message{Role: message.User}

	history := recoverToolHistory([]message.Message{assistant, saved, followUp})
	require.Len(t, history, 3)
	require.Equal(t, followUp, history[2])
	require.Len(t, history[0].ToolCalls(), 2)
	require.Equal(t, "{}", history[0].ToolCalls()[1].Input)
	require.Equal(t, "{", assistant.ToolCalls()[1].Input)
	results := history[1].ToolResults()
	require.Len(t, results, 2)
	require.Equal(t, saved.ToolResults()[0], results[0])
	require.Equal(t, "interrupted", results[1].ToolCallID)
	require.True(t, results[1].IsError)
	require.Contains(t, results[1].Content, "outcome is unknown")
}
