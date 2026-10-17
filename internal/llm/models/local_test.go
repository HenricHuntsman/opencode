package models

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustParse(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return u
}

func resetLocalState(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		for id, m := range SupportedModels {
			if m.Provider == ProviderLocal {
				delete(SupportedModels, id)
			}
		}
		delete(ProviderPopularity, ProviderLocal)
		viper.Reset()
	})
}

func fakeOllama(t *testing.T, withShow bool) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"id": "qwen2.5-coder:7b", "object": "model"}},
		})
	})
	if withShow {
		mux.HandleFunc("/api/show", func(w http.ResponseWriter, r *http.Request) {
			var req map[string]string
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req["model"] != "qwen2.5-coder:7b" {
				http.NotFound(w, r)
				return
			}
			json.NewEncoder(w).Encode(map[string]any{
				"model_info": map[string]any{
					"general.architecture": "qwen2",
					"qwen2.context_length": 32768,
				},
			})
		})
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestInitLocalModels_OllamaContextLength(t *testing.T) {
	resetLocalState(t)
	srv := fakeOllama(t, true)

	initLocalModels(srv.URL + "/v1")

	m, ok := SupportedModels["local.qwen2.5-coder:7b"]
	require.True(t, ok, "model should be registered")
	assert.Equal(t, ProviderLocal, m.Provider)
	assert.Equal(t, "qwen2.5-coder:7b", m.APIModel)
	assert.EqualValues(t, 32768, m.ContextWindow)
	assert.EqualValues(t, "local.qwen2.5-coder:7b", viper.Get("agents.coder.model"))
}

func TestInitLocalModels_FallsBackWithoutShowEndpoint(t *testing.T) {
	resetLocalState(t)
	srv := fakeOllama(t, false)

	initLocalModels(srv.URL + "/v1")

	m, ok := SupportedModels["local.qwen2.5-coder:7b"]
	require.True(t, ok)
	assert.EqualValues(t, 4096, m.ContextWindow)
}

func TestInitLocalModels_DeadEndpointReturnsQuickly(t *testing.T) {
	resetLocalState(t)
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) })

	orig := localHTTPClient.Timeout
	localHTTPClient.Timeout = 200 * time.Millisecond
	t.Cleanup(func() { localHTTPClient.Timeout = orig })

	start := time.Now()
	initLocalModels(srv.URL + "/v1")
	elapsed := time.Since(start)

	assert.Less(t, elapsed, 2*time.Second, "discovery must time out instead of hanging")
	for _, m := range SupportedModels {
		assert.NotEqual(t, ProviderLocal, m.Provider)
	}
}

func TestInitLocalModels_IgnoresEmptyAndInvalidEndpoints(t *testing.T) {
	resetLocalState(t)
	initLocalModels("")
	initLocalModels("http://[::1")
	for _, m := range SupportedModels {
		assert.NotEqual(t, ProviderLocal, m.Provider)
	}
}

func TestOllamaContextLength(t *testing.T) {
	srv := fakeOllama(t, true)
	u := mustParse(t, srv.URL+"/v1")

	assert.EqualValues(t, 32768, ollamaContextLength(*u, "qwen2.5-coder:7b"))
	assert.EqualValues(t, 0, ollamaContextLength(*u, "unknown-model"))
}
