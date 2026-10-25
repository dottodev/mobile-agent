package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"mobile-agent/internal/types"
)

func TestOpenAIComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected auth %s", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		var req map[string]any
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatal(err)
		}
		if req["model"] != "test-model" {
			t.Fatalf("unexpected model %v", req["model"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"chat1","choices":[{"index":0,"message":{"role":"assistant","content":"hello"},"finish_reason":"stop"}]}`)
	}))
	defer srv.Close()

	p, err := NewOpenAI(OpenAIConfig{BaseURL: srv.URL, APIKey: "test-key", Model: "test-model"}, "default")
	if err != nil {
		t.Fatal(err)
	}

	resp, err := p.Complete(context.Background(), types.CompletionRequest{
		Model:    "test-model",
		Messages: []types.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Message.Content != "hello" {
		t.Fatalf("unexpected content %q", resp.Message.Content)
	}
}

func TestOpenAIToolCalls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"chat2","choices":[{"index":0,"message":{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"bash","arguments":"{\"command\":\"pwd\"}"}}]},"finish_reason":"tool_calls"}]}`)
	}))
	defer srv.Close()

	p, err := NewOpenAI(OpenAIConfig{BaseURL: srv.URL, APIKey: "test-key", Model: "test-model"}, "default")
	if err != nil {
		t.Fatal(err)
	}

	resp, err := p.Complete(context.Background(), types.CompletionRequest{Model: "test-model"})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Message.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.Message.ToolCalls))
	}
	if resp.Message.ToolCalls[0].Name != "bash" {
		t.Fatalf("unexpected tool name %s", resp.Message.ToolCalls[0].Name)
	}
}
