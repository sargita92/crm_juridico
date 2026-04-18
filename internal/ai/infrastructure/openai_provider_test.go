package infrastructure

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
)

func newTestLogger(t *testing.T) *zap.Logger {
	t.Helper()
	log, err := zap.NewDevelopment()
	require.NoError(t, err)
	return log
}

func validRequest(t *testing.T) *domain.AIRequest {
	t.Helper()
	msg, err := domain.NewAIMessage(domain.RoleUser, "Hello")
	require.NoError(t, err)
	req, err := domain.NewAIRequest("You are helpful.", []domain.AIMessage{msg}, "openai", "gpt-4o", 0.7, 256)
	require.NoError(t, err)
	return req
}

func TestOpenAIProvider_Name(t *testing.T) {
	p := NewOpenAIProvider("key", "", newTestLogger(t))
	assert.Equal(t, "openai", p.Name())
}

func TestOpenAIProvider_GenerateResponse_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message":       map[string]string{"role": "assistant", "content": "Hello, world!"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]int{
				"prompt_tokens":     10,
				"completion_tokens": 5,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := NewOpenAIProvider("test-key", srv.URL, newTestLogger(t))
	resp, err := p.GenerateResponse(t.Context(), validRequest(t))

	require.NoError(t, err)
	assert.Equal(t, "Hello, world!", resp.Content)
	assert.Equal(t, "stop", resp.FinishReason)
	assert.Equal(t, 10, resp.PromptTokens)
	assert.Equal(t, 5, resp.CompletionTokens)
}

func TestOpenAIProvider_GenerateResponse_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		resp := map[string]any{
			"error": map[string]string{
				"message": "Rate limit exceeded",
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := NewOpenAIProvider("test-key", srv.URL, newTestLogger(t))
	resp, err := p.GenerateResponse(t.Context(), validRequest(t))

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "429")
	assert.Contains(t, err.Error(), "Rate limit exceeded")
}

func TestOpenAIProvider_SystemPrompt(t *testing.T) {
	var capturedBody openAIRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message":       map[string]string{"role": "assistant", "content": "OK"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]int{
				"prompt_tokens":     5,
				"completion_tokens": 1,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := NewOpenAIProvider("test-key", srv.URL, newTestLogger(t))
	_, err := p.GenerateResponse(t.Context(), validRequest(t))
	require.NoError(t, err)

	require.NotEmpty(t, capturedBody.Messages, "messages must not be empty")
	first := capturedBody.Messages[0]
	assert.Equal(t, "system", first.Role, "first message role must be 'system'")
	assert.Equal(t, "You are helpful.", first.Content)
}

func TestOpenAIProvider_UsesMaxCompletionTokensField(t *testing.T) {
	var rawBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&rawBody))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "ok"}, "finish_reason": "stop"},
			},
			"usage": map[string]int{"prompt_tokens": 1, "completion_tokens": 1},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := NewOpenAIProvider("test-key", srv.URL, newTestLogger(t))
	_, err := p.GenerateResponse(t.Context(), validRequest(t))
	require.NoError(t, err)

	_, hasOld := rawBody["max_tokens"]
	_, hasNew := rawBody["max_completion_tokens"]
	assert.False(t, hasOld, "must not send 'max_tokens' (rejected by newer models like gpt-5.* and o-series)")
	assert.True(t, hasNew, "must send 'max_completion_tokens'")
}
