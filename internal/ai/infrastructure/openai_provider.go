package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
)

const defaultOpenAIBaseURL = "https://api.openai.com/v1/chat/completions"

// OpenAIProvider implements domain.AIProvider using the OpenAI HTTP API.
type OpenAIProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
	log     *zap.Logger
}

// NewOpenAIProvider creates an OpenAIProvider. If baseURL is empty, the default
// OpenAI completions endpoint is used.
func NewOpenAIProvider(apiKey, baseURL string, log *zap.Logger) *OpenAIProvider {
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}
	return &OpenAIProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 60 * time.Second},
		log:     log,
	}
}

// Name returns the provider identifier.
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// GenerateResponse sends a request to the OpenAI API and returns the response.
func (p *OpenAIProvider) GenerateResponse(ctx context.Context, req *domain.AIRequest) (*domain.AIResponse, error) {
	messages := p.buildMessages(req)
	messages = append(messages, p.buildToolResultMessages(req.ToolResults)...)

	body := openAIRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Tools:       p.buildOpenAITools(req.Tools),
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("openai: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("openai: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	start := time.Now()
	resp, err := p.client.Do(httpReq)
	if err != nil {
		p.log.Error("openai: request failed",
			zap.Error(err),
			zap.Duration("duration", time.Since(start)),
		)
		return nil, fmt.Errorf("openai: http request: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(start)
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("openai: read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr openAIError
		if jsonErr := json.Unmarshal(respBytes, &apiErr); jsonErr == nil && apiErr.Error.Message != "" {
			p.log.Error("openai: API error",
				zap.Int("status", resp.StatusCode),
				zap.String("error", apiErr.Error.Message),
				zap.Duration("duration", duration),
			)
			return nil, fmt.Errorf("openai: API error (status %d): %s", resp.StatusCode, apiErr.Error.Message)
		}
		p.log.Error("openai: non-200 response",
			zap.Int("status", resp.StatusCode),
			zap.Duration("duration", duration),
		)
		return nil, fmt.Errorf("openai: unexpected status %d", resp.StatusCode)
	}

	var apiResp openAIResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("openai: unmarshal response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("openai: no choices in response")
	}

	choice := apiResp.Choices[0]
	p.log.Info("openai: response received",
		zap.String("model", req.Model),
		zap.Int("prompt_tokens", apiResp.Usage.PromptTokens),
		zap.Int("completion_tokens", apiResp.Usage.CompletionTokens),
		zap.Duration("duration", duration),
	)

	toolCalls := p.parseToolCalls(apiResp.Choices)

	return &domain.AIResponse{
		Content:          choice.Message.Content,
		FinishReason:     choice.FinishReason,
		PromptTokens:     apiResp.Usage.PromptTokens,
		CompletionTokens: apiResp.Usage.CompletionTokens,
		ToolCalls:        toolCalls,
	}, nil
}

// buildMessages converts an AIRequest into the OpenAI messages slice, prepending
// the system prompt when present.
func (p *OpenAIProvider) buildMessages(req *domain.AIRequest) []openAIMessage {
	var messages []openAIMessage
	if req.SystemPrompt != "" {
		messages = append(messages, openAIMessage{
			Role:    string(domain.RoleSystem),
			Content: req.SystemPrompt,
		})
	}
	for _, m := range req.Messages {
		messages = append(messages, openAIMessage{
			Role:    string(m.Role),
			Content: m.Content,
		})
	}
	return messages
}

// buildOpenAITools converts domain ToolDefinitions to OpenAI function calling format.
func (p *OpenAIProvider) buildOpenAITools(tools []domain.ToolDefinition) []openAITool {
	if len(tools) == 0 {
		return nil
	}
	result := make([]openAITool, len(tools))
	for i, td := range tools {
		properties := make(map[string]interface{})
		var required []string

		for name, param := range td.Parameters {
			prop := map[string]interface{}{
				"type":        param.Type,
				"description": param.Description,
			}
			if len(param.Enum) > 0 {
				prop["enum"] = param.Enum
			}
			properties[name] = prop
			if param.Required {
				required = append(required, name)
			}
		}

		params := map[string]interface{}{
			"type":       "object",
			"properties": properties,
		}
		if len(required) > 0 {
			params["required"] = required
		}

		result[i] = openAITool{
			Type: "function",
			Function: openAIFunction{
				Name:        td.Name,
				Description: td.Description,
				Parameters:  params,
			},
		}
	}
	return result
}

// parseToolCalls extracts domain ToolCalls from OpenAI response choices.
func (p *OpenAIProvider) parseToolCalls(choices []openAIChoice) []domain.ToolCall {
	if len(choices) == 0 {
		return nil
	}
	msg := choices[0].Message
	if len(msg.ToolCalls) == 0 {
		return nil
	}

	calls := make([]domain.ToolCall, len(msg.ToolCalls))
	for i, tc := range msg.ToolCalls {
		var args map[string]interface{}
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		if args == nil {
			args = make(map[string]interface{})
		}
		calls[i] = domain.ToolCall{
			ID:        tc.ID,
			ToolName:  tc.Function.Name,
			Arguments: args,
		}
	}
	return calls
}

// buildToolResultMessages converts domain ToolResults to OpenAI tool messages.
func (p *OpenAIProvider) buildToolResultMessages(results []domain.ToolResult) []openAIMessage {
	msgs := make([]openAIMessage, len(results))
	for i, r := range results {
		msgs[i] = openAIMessage{
			Role:       "tool",
			ToolCallID: r.ToolCallID,
			Content:    r.Content,
		}
	}
	return msgs
}

// — internal JSON structs —

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_completion_tokens"`
	Tools       []openAITool    `json:"tools,omitempty"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type openAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openAIFunctionCall `json:"function"`
}

type openAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIResponse struct {
	Choices []openAIChoice `json:"choices"`
	Usage   openAIUsage    `json:"usage"`
}

type openAIChoice struct {
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

type openAIError struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}
