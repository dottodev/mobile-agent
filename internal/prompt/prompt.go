package prompt

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"mobile-agent/internal/types"
)

type Part struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	File    string `json:"file"`
}

type Stack struct {
	Name  string `json:"name"`
	Parts []Part `json:"parts"`
}

var BuiltIn = map[string]*Stack{
	"build": {
		Name: "build",
		Parts: []Part{
			{Role: "system", Content: "You are a capable coding and automation agent running in a terminal. You can use shell, file, grep, glob, git, and web tools. Make small, precise changes. Ask before destructive actions unless unrestricted mode is enabled."},
		},
	},
	"review": {
		Name: "review",
		Parts: []Part{
			{Role: "system", Content: "You are a code review agent. Analyze code for correctness, security, performance, and clarity. Do not make changes unless asked."},
		},
	},
}

func LoadStack(dir, name string) (*Stack, error) {
	path := filepath.Join(dir, name+".json")
	if data, err := os.ReadFile(path); err == nil {
		var s Stack
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, err
		}
		s.Name = name
		return &s, nil
	}
	if s, ok := BuiltIn[name]; ok {
		copy := *s
		return &copy, nil
	}
	return nil, os.ErrNotExist
}

func List(dir string) ([]string, error) {
	set := map[string]bool{}
	for name := range BuiltIn {
		set[name] = true
	}
	if entries, err := os.ReadDir(dir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			set[strings.TrimSuffix(e.Name(), ".json")] = true
		}
	}
	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

func Compose(s *Stack, extraSystem string) []types.Message {
	msgs := make([]types.Message, 0, len(s.Parts)+1)
	for _, p := range s.Parts {
		content := p.Content
		if p.File != "" {
			if data, err := os.ReadFile(p.File); err == nil {
				content = string(data)
			}
		}
		msgs = append(msgs, types.Message{Role: p.Role, Content: content})
	}
	if extraSystem != "" {
		msgs = append(msgs, types.Message{Role: "system", Content: extraSystem})
	}
	return msgs
}
