package agent

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

type inputMCPClient struct {
	called  bool
	closed  bool
	request mcp.CallToolRequest
}

func (c *inputMCPClient) Initialize(context.Context, mcp.InitializeRequest) (*mcp.InitializeResult, error) {
	return &mcp.InitializeResult{}, nil
}
func (c *inputMCPClient) ListTools(context.Context, mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	return &mcp.ListToolsResult{}, nil
}
func (c *inputMCPClient) CallTool(_ context.Context, r mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	c.called = true
	c.request = r
	return &mcp.CallToolResult{Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "ok"}}}, nil
}
func (c *inputMCPClient) Close() error { c.closed = true; return nil }

func TestRunToolEmptyInput(t *testing.T) {
	for _, input := range []string{"", " ", "\n\t", "{}"} {
		t.Run(input, func(t *testing.T) {
			c := &inputMCPClient{}
			response, err := runTool(context.Background(), c, "no-args", input)
			require.NoError(t, err)
			require.False(t, response.IsError)
			require.True(t, c.called)
			require.True(t, c.closed)
			require.Equal(t, "no-args", c.request.Params.Name)
			require.NotNil(t, c.request.Params.Arguments)
			require.Empty(t, c.request.Params.Arguments)
		})
	}
}
func TestRunToolPreservesParametersAndRejectsInvalidJSON(t *testing.T) {
	c := &inputMCPClient{}
	response, err := runTool(context.Background(), c, "search", `{"query":"hello"}`)
	require.NoError(t, err)
	require.False(t, response.IsError)
	require.Equal(t, map[string]any{"query": "hello"}, c.request.Params.Arguments)
	c = &inputMCPClient{}
	response, err = runTool(context.Background(), c, "search", `{broken`)
	require.NoError(t, err)
	require.True(t, response.IsError)
	require.False(t, c.called)
	require.True(t, c.closed)
}
