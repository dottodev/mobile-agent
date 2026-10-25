package types

import (
	"encoding/json"
	"time"
)

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type CompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Tools       []ToolDef `json:"tools,omitempty"`
	ToolChoice  string    `json:"tool_choice,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

type CompletionResponse struct {
	Message Message         `json:"message"`
	Raw     json.RawMessage `json:"-"`
}

type Session struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Provider string    `json:"provider"`
	Model    string    `json:"model"`
	Stack    string    `json:"stack"`
	CWD      string    `json:"cwd"`
	Messages []Message `json:"messages"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
}
