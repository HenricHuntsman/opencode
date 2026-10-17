package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/opencode-ai/opencode/internal/skills"
)

const SkillToolName = "skill"

type SkillParams struct {
	Name string `json:"name"`
}

type skillTool struct {
	skills []skills.Skill
}

// NewSkillTool returns a tool that loads SKILL.md instructions on demand.
func NewSkillTool(available []skills.Skill) BaseTool {
	return &skillTool{skills: available}
}

func (s *skillTool) Info() ToolInfo {
	var b strings.Builder
	b.WriteString("Load a skill: reusable, specialised instructions for a task. Call it with the skill name when a listed skill matches the work, then follow the returned instructions.\n\n<available_skills>\n")
	for _, sk := range s.skills {
		if !sk.AutoInvoke() {
			continue
		}
		fmt.Fprintf(&b, "  <skill>\n    <name>%s</name>\n    <description>%s</description>\n  </skill>\n", sk.Name, sk.Description)
	}
	b.WriteString("</available_skills>")

	return ToolInfo{
		Name:        SkillToolName,
		Description: b.String(),
		Parameters: map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Name of the skill to load",
			},
		},
		Required: []string{"name"},
	}
}

func (s *skillTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params SkillParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}
	if params.Name == "" {
		return NewTextErrorResponse("name is required"), nil
	}

	for _, sk := range s.skills {
		if sk.Name == params.Name {
			return NewTextResponse(fmt.Sprintf("<skill name=%q>\n%s\n</skill>", sk.Name, sk.Body)), nil
		}
	}

	names := make([]string, 0, len(s.skills))
	for _, sk := range s.skills {
		names = append(names, sk.Name)
	}
	return NewTextErrorResponse(fmt.Sprintf("skill %q not found. Available skills: %s", params.Name, strings.Join(names, ", "))), nil
}
