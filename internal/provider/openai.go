package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"mobile-agent/internal/types"
)

type OpenAIConfig struct {
	BaseURL string
	APIKey  string
	Model   string
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openAIFunctionCall `json:"function"`
}

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Tools       []types.ToolDef `json:"tools,omitempty"`
	ToolChoice  string          `json:"tool_choice,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

type openAIResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Index        int           `json:"index"`
		Message      openAIMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
}

type OpenAIProvider struct {
	client  *http.Client
	baseURL string
	apiKey  string
	model   string
}

func NewOpenAI(cfg OpenAIConfig, defaultModel string) (*OpenAIProvider, error) {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	model := cfg.Model
	if model == "" {
		model = defaultModel
	}
	if model == "" {
		return nil, errors.New("missing model")
	}
	return &OpenAIProvider{
		client:  &http.Client{Timeout: 120 * time.Second},
		baseURL: baseURL,
		apiKey:  cfg.APIKey,
		model:   model,
	}, nil
}

func (p *OpenAIProvider) Name() string {
	return "openai-compatible"
}

func (p *OpenAIProvider) Complete(ctx context.Context, req types.CompletionRequest) (*types.CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	msg := openAIRequest{
		Model:       model,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      false,
		ToolChoice:  "auto",
	}

	for _, m := range req.Messages {
		om := openAIMessage{Role: m.Role, Content: m.Content, ToolCallID: m.ToolCallID}
		for _, tc := range m.ToolCalls {
			om.ToolCalls = append(om.ToolCalls, openAIToolCall{
				ID:       tc.ID,
				Type:     "function",
				Function: openAIFunctionCall{Name: tc.Name, Arguments: tc.Arguments},
			})
		}
		msg.Messages = append(msg.Messages, om)
	}

	if len(req.Tools) == 0 {
		msg.ToolChoice = "none"
	} else {
		msg.ToolChoice = "auto"
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	reqURL := p.baseURL + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("provider error: %s: %s", resp.Status, string(body))
	}

	var parsed openAIResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("provider parse error: %w: %s", err, string(body))
	}

	if len(parsed.Choices) == 0 {
		return nil, errors.New("no completion choices")
	}

	out := types.CompletionResponse{
		Message: types.Message{
			Role:       parsed.Choices[0].Message.Role,
			Content:    parsed.Choices[0].Message.Content,
			ToolCallID: parsed.Choices[0].Message.ToolCallID,
		},
	}
	for _, tc := range parsed.Choices[0].Message.ToolCalls {
		out.Message.ToolCalls = append(out.Message.ToolCalls, types.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}
	out.Raw = body
	return &out, nil
}
