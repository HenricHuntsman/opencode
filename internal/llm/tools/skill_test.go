package tools

import (
	"context"
	"testing"

	"github.com/opencode-ai/opencode/internal/skills"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testSkills() []skills.Skill {
	return []skills.Skill{
		{Name: "git-release", Description: "Create releases", Body: "Draft notes."},
		{Name: "hidden", Description: "Manual only", Body: "Secret.", Metadata: map[string]string{"opencode/autoinvoke": "false"}},
	}
}

func runSkill(t *testing.T, tool BaseTool, input string) ToolResponse {
	t.Helper()
	resp, err := tool.Run(context.Background(), ToolCall{Name: SkillToolName, Input: input})
	require.NoError(t, err)
	return resp
}

func TestSkillTool_Info(t *testing.T) {
	info := NewSkillTool(testSkills()).Info()
	assert.Equal(t, SkillToolName, info.Name)
	assert.Contains(t, info.Description, "<name>git-release</name>")
	assert.Contains(t, info.Description, "Create releases")
	assert.NotContains(t, info.Description, "hidden")
	assert.Equal(t, []string{"name"}, info.Required)
}

func TestSkillTool_Run(t *testing.T) {
	tool := NewSkillTool(testSkills())

	t.Run("loads a skill by name", func(t *testing.T) {
		resp := runSkill(t, tool, `{"name":"git-release"}`)
		assert.False(t, resp.IsError)
		assert.Contains(t, resp.Content, "Draft notes.")
	})

	t.Run("loads a non-autoinvoke skill by exact name", func(t *testing.T) {
		resp := runSkill(t, tool, `{"name":"hidden"}`)
		assert.False(t, resp.IsError)
		assert.Contains(t, resp.Content, "Secret.")
	})

	t.Run("unknown skill lists alternatives", func(t *testing.T) {
		resp := runSkill(t, tool, `{"name":"nope"}`)
		assert.True(t, resp.IsError)
		assert.Contains(t, resp.Content, "git-release")
	})

	t.Run("missing name", func(t *testing.T) {
		resp := runSkill(t, tool, `{}`)
		assert.True(t, resp.IsError)
	})

	t.Run("invalid json", func(t *testing.T) {
		resp := runSkill(t, tool, `not json`)
		assert.True(t, resp.IsError)
	})

	t.Run("empty skill set", func(t *testing.T) {
		empty := NewSkillTool(nil)
		assert.NotContains(t, empty.Info().Description, "<skill>")
		assert.True(t, runSkill(t, empty, `{"name":"x"}`).IsError)
	})
}
