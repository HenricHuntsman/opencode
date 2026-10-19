package provider

import (
	"encoding/json"
	"testing"

	"github.com/opencode-ai/opencode/internal/llm/models"
	"github.com/stretchr/testify/require"
)

func TestReasoningEffortEndpointCompatibility(t *testing.T) {
	t.Setenv("LOCAL_ENDPOINT", "http://127.0.0.1:11434/v1")
	model := models.Model{APIModel: "fixture", CanReason: true}
	for _, name := range []models.ModelProvider{models.ProviderLocal, models.ProviderOpenAI} {
		t.Run(string(name), func(t *testing.T) {
			p, err := NewProvider(name, WithModel(model), WithMaxTokens(512))
			require.NoError(t, err)
			client := p.(*baseProvider[OpenAIClient]).client.(*openaiClient)
			data, err := json.Marshal(client.preparedParams(nil, nil))
			require.NoError(t, err)
			var fields map[string]any
			require.NoError(t, json.Unmarshal(data, &fields))
			if name == models.ProviderLocal {
				require.NotContains(t, fields, "reasoning_effort")
			} else {
				require.Equal(t, "medium", fields["reasoning_effort"])
			}
			require.EqualValues(t, 512, fields["max_completion_tokens"])
		})
	}
}
