package models

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/spf13/viper"
)

const (
	ProviderLocal ModelProvider = "local"

	localModelsPath        = "v1/models"
	lmStudioBetaModelsPath = "api/v0/models"
	ollamaShowPath         = "api/show"
)

// A dead endpoint must not stall startup, since discovery runs in init().
var localHTTPClient = &http.Client{Timeout: 1500 * time.Millisecond}

func init() {
	initLocalModels(os.Getenv("LOCAL_ENDPOINT"))
}

func initLocalModels(endpoint string) {
	if endpoint == "" {
		return
	}

	localEndpoint, err := url.Parse(endpoint)
	if err != nil {
		logging.Debug("Failed to parse local endpoint",
			"error", err,
			"endpoint", endpoint,
		)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	load := func(u url.URL, path string) []localModel {
		u.Path = path
		return listLocalModelsContext(ctx, u.String())
	}

	models := load(*localEndpoint, lmStudioBetaModelsPath)

	if len(models) == 0 {
		models = load(*localEndpoint, localModelsPath)
	}

	if len(models) == 0 {
		logging.Debug("No local models found",
			"endpoint", endpoint,
		)
		return
	}

	for i := range models {
		if models[i].LoadedContextLength == 0 {
			models[i].LoadedContextLength = ollamaContextLengthContext(ctx, *localEndpoint, models[i].ID)
		}
	}

	loadLocalModels(models)

	viper.SetDefault("providers.local.apiKey", "dummy")
	ProviderPopularity[ProviderLocal] = 0
}

// ollamaContextLength asks an Ollama server for a model's context length.
// It returns 0 for any other server or on any failure.
func ollamaContextLength(endpoint url.URL, modelID string) int64 {
	return ollamaContextLengthContext(context.Background(), endpoint, modelID)
}

func ollamaContextLengthContext(ctx context.Context, endpoint url.URL, modelID string) int64 {
	endpoint.Path = ollamaShowPath
	body, err := json.Marshal(map[string]string{"model": modelID})
	if err != nil {
		return 0
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return 0
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := localHTTPClient.Do(req)
	if err != nil {
		return 0
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return 0
	}

	var show struct {
		ModelInfo map[string]any `json:"model_info"`
	}
	if err := json.NewDecoder(res.Body).Decode(&show); err != nil {
		return 0
	}
	for key, value := range show.ModelInfo {
		if strings.HasSuffix(key, ".context_length") {
			if n, ok := value.(float64); ok && n > 0 {
				return int64(n)
			}
		}
	}
	return 0
}

type localModelList struct {
	Data []localModel `json:"data"`
}

type localModel struct {
	ID                  string `json:"id"`
	Object              string `json:"object"`
	Type                string `json:"type"`
	Publisher           string `json:"publisher"`
	Arch                string `json:"arch"`
	CompatibilityType   string `json:"compatibility_type"`
	Quantization        string `json:"quantization"`
	State               string `json:"state"`
	MaxContextLength    int64  `json:"max_context_length"`
	LoadedContextLength int64  `json:"loaded_context_length"`
}

func listLocalModels(modelsEndpoint string) []localModel {
	return listLocalModelsContext(context.Background(), modelsEndpoint)
}

func listLocalModelsContext(ctx context.Context, modelsEndpoint string) []localModel {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsEndpoint, nil)
	if err != nil {
		return nil
	}
	res, err := localHTTPClient.Do(req)
	if err != nil {
		logging.Debug("Failed to list local models",
			"error", err,
			"endpoint", modelsEndpoint,
		)
		return []localModel{}
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		logging.Debug("Failed to list local models",
			"status", res.StatusCode,
			"endpoint", modelsEndpoint,
		)
		return []localModel{}
	}

	var modelList localModelList
	if err = json.NewDecoder(res.Body).Decode(&modelList); err != nil {
		logging.Debug("Failed to list local models",
			"error", err,
			"endpoint", modelsEndpoint,
		)
		return []localModel{}
	}

	var supportedModels []localModel
	for _, model := range modelList.Data {
		if strings.HasSuffix(modelsEndpoint, lmStudioBetaModelsPath) {
			if model.Object != "model" || model.Type != "llm" {
				logging.Debug("Skipping unsupported LMStudio model",
					"endpoint", modelsEndpoint,
					"id", model.ID,
					"object", model.Object,
					"type", model.Type,
				)

				continue
			}
		}

		supportedModels = append(supportedModels, model)
	}

	return supportedModels
}

func loadLocalModels(models []localModel) {
	for i, m := range models {
		model := convertLocalModel(m)
		SupportedModels[model.ID] = model

		if i == 0 || m.State == "loaded" {
			viper.SetDefault("agents.coder.model", model.ID)
			viper.SetDefault("agents.summarizer.model", model.ID)
			viper.SetDefault("agents.task.model", model.ID)
			viper.SetDefault("agents.title.model", model.ID)
		}
	}
}

func convertLocalModel(model localModel) Model {
	return Model{
		ID:                  ModelID("local." + model.ID),
		Name:                friendlyModelName(model.ID),
		Provider:            ProviderLocal,
		APIModel:            model.ID,
		ContextWindow:       cmp.Or(model.LoadedContextLength, 4096),
		DefaultMaxTokens:    cmp.Or(model.LoadedContextLength, 4096),
		CanReason:           true,
		SupportsAttachments: true,
	}
}

var modelInfoRegex = regexp.MustCompile(`(?i)^([a-z0-9]+)(?:[-_]?([rv]?\d[\.\d]*))?(?:[-_]?([a-z]+))?.*`)

func friendlyModelName(modelID string) string {
	mainID := modelID
	tag := ""

	if slash := strings.LastIndex(mainID, "/"); slash != -1 {
		mainID = mainID[slash+1:]
	}

	if at := strings.Index(modelID, "@"); at != -1 {
		mainID = modelID[:at]
		tag = modelID[at+1:]
	}

	match := modelInfoRegex.FindStringSubmatch(mainID)
	if match == nil {
		return modelID
	}

	capitalize := func(s string) string {
		if s == "" {
			return ""
		}
		runes := []rune(s)
		runes[0] = unicode.ToUpper(runes[0])
		return string(runes)
	}

	family := capitalize(match[1])
	version := ""
	label := ""

	if len(match) > 2 && match[2] != "" {
		version = strings.ToUpper(match[2])
	}

	if len(match) > 3 && match[3] != "" {
		label = capitalize(match[3])
	}

	var parts []string
	if family != "" {
		parts = append(parts, family)
	}
	if version != "" {
		parts = append(parts, version)
	}
	if label != "" {
		parts = append(parts, label)
	}
	if tag != "" {
		parts = append(parts, tag)
	}

	return strings.Join(parts, " ")
}
