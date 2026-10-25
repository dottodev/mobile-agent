package agent

import (
	"bytes"
	"context"
	"testing"

	"mobile-agent/internal/config"
	"mobile-agent/internal/store"
	"mobile-agent/internal/tool"
	"mobile-agent/internal/types"
)

type mockProvider struct {
	resp types.CompletionResponse
	err  error
}

func (m mockProvider) Name() string {
	return "mock"
}

func (m mockProvider) Complete(ctx context.Context, req types.CompletionRequest) (*types.CompletionResponse, error) {
	return &m.resp, m.err
}

type mockPrompter struct{}

func (mockPrompter) Confirm(prompt string) (bool, error) {
	return true, nil
}

func TestAgentRunNoTools(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.PromptsDir = dir

	a := &Agent{
		store:    store.New(dir),
		provider: mockProvider{resp: types.CompletionResponse{Message: types.Message{Role: "assistant", Content: "ok"}}},
		tools:    map[string]tool.Tool{},
		prompter: mockPrompter{},
		out:      &bytes.Buffer{},
		maxSteps: 5,
	}

	if err := a.NewSession("mock", "model"); err != nil {
		t.Fatal(err)
	}
	if err := a.Run(context.Background(), "hi"); err != nil {
		t.Fatal(err)
	}
	if len(a.session.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(a.session.Messages))
	}
}
