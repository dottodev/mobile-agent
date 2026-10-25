package tool

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"mobile-agent/internal/types"
)

type Input struct {
	ID        string
	Name      string
	Arguments string
	SessionID string
	CWD       string
}

type Result struct {
	ID     string
	Output string
	Error  bool
}

type Tool interface {
	Name() string
	Description() string
	Parameters() types.ToolDef
	Call(ctx context.Context, in Input) (*Result, error)
}

type ShellTool struct{}

func (ShellTool) Name() string { return "bash" }

func (ShellTool) Description() string { return "Execute a shell command" }

func (ShellTool) Parameters() types.ToolDef {
	return types.ToolDef{
		Name:        "bash",
		Description: "Execute a shell command",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{"type": "string", "description": "Shell command to execute"},
				"cwd":     map[string]any{"type": "string", "description": "Working directory"},
			},
			"required": []string{"command"},
		},
	}
}

func (ShellTool) Call(ctx context.Context, in Input) (*Result, error) {
	var args struct {
		Command string `json:"command"`
		CWD     string `json:"cwd"`
	}
	if err := json.Unmarshal([]byte(in.Arguments), &args); err != nil {
		return &Result{ID: in.ID, Output: err.Error(), Error: true}, nil
	}
	cmd := exec.CommandContext(ctx, "sh", "-c", args.Command)
	if args.CWD != "" {
		cmd.Dir = args.CWD
	} else if in.CWD != "" {
		cmd.Dir = in.CWD
	}
	cmd.Env = os.Environ()
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return &Result{ID: in.ID, Output: out.String(), Error: err != nil}, nil
}

type FileTool struct{}

func (FileTool) Name() string { return "file" }

func (FileTool) Description() string { return "Read, write, or edit files" }

func (FileTool) Parameters() types.ToolDef {
	return types.ToolDef{
		Name:        "file",
		Description: "Read, write, or edit files",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"op":      map[string]any{"type": "string", "description": "read, write, or edit"},
				"path":    map[string]any{"type": "string", "description": "File path"},
				"content": map[string]any{"type": "string", "description": "File content for write or edit"},
				"old":     map[string]any{"type": "string", "description": "Exact text to replace"},
				"new":     map[string]any{"type": "string", "description": "Replacement text"},
			},
			"required": []string{"op", "path"},
		},
	}
}

func (FileTool) Call(ctx context.Context, in Input) (*Result, error) {
	_ = ctx
	var args struct {
		Op      string `json:"op"`
		Path    string `json:"path"`
		Content string `json:"content"`
		Old     string `json:"old"`
		New     string `json:"new"`
	}
	if err := json.Unmarshal([]byte(in.Arguments), &args); err != nil {
		return &Result{ID: in.ID, Output: err.Error(), Error: true}, nil
	}
	path := args.Path
	if !filepath.IsAbs(path) && in.CWD != "" {
		path = filepath.Join(in.CWD, path)
	}
	path = filepath.Clean(path)

	switch args.Op {
	case "read":
		data, err := os.ReadFile(path)
		if err != nil {
			return &Result{ID: in.ID, Output: err.Error(), Error: true}, nil
		}
		return &Result{ID: in.ID, Output: string(data)}, nil
	case "write":
		if err := os.WriteFile(path, []byte(args.Content), 0644); err != nil {
			return &Result{ID: in.ID, Output: err.Error(), Error: true}, nil
		}
		return &Result{ID: in.ID, Output: "wrote " + path}, nil
	case "edit":
		data, err := os.ReadFile(path)
		if err != nil {
			return &Result{ID: in.ID, Output: err.Error(), Error: true}, nil
		}
		content := string(data)
		if !strings.Contains(content, args.Old) {
			return &Result{ID: in.ID, Output: "old text not found", Error: true}, nil
		}
		content = strings.Replace(content, args.Old, args.New, 1)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return &Result{ID: in.ID, Output: err.Error(), Error: true}, nil
		}
		return &Result{ID: in.ID, Output: "edited " + path}, nil
	default:
		return &Result{ID: in.ID, Output: "unknown op " + args.Op, Error: true}, nil
	}
}

type GrepTool struct{}

func (GrepTool) Name() string { return "grep" }

func (GrepTool) Description() string { return "Search file contents" }

func (GrepTool) Parameters() types.ToolDef {
	return types.ToolDef{
		Name:        "grep",
		Description: "Search file contents",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{"type": "string", "description": "Regex pattern"},
				"path":    map[string]any{"type": "string", "description": "Directory or file to search"},
			},
			"required": []string{"pattern"},
		},
	}
}

func (GrepTool) Call(ctx context.Context, in Input) (*Result, error) {
	_ = ctx
	var args struct {
		Pattern string `json:"pattern"`
		Path    string `json:"path"`
	}
	if err := json.Unmarshal([]byte(in.Arguments), &args); err != nil {
		return &Result{ID: in.ID, Output: err.Error(), Error: true}, nil
	}
	root := args.Path
	if root == "" {
		root = in.CWD
	}
	if root == "" {
		root = "."
	}

	var out strings.Builder
	count := 0
	pattern := args.Pattern

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "node_modules" {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		scanner := bufio.NewScanner(bytes.NewReader(data))
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			matched, err := regexp.MatchString(pattern, line)
			if err != nil || !matched {
				continue
			}
			rel, _ := filepath.Rel(root, path)
			out.WriteString(fmt.Sprintf("%s:%d: %s\n", rel, lineNum, line))
			count++
			if count >= 1000 {
				return nil
			}
		}
		return nil
	})
	if err != nil {
		return &Result{ID: in.ID, Output: err.Error(), Error: true}, nil
	}
	return &Result{ID: in.ID, Output: out.String()}, nil
}

type GlobTool struct{}

func (GlobTool) Name() string { return "glob" }

func (GlobTool) Description() string { return "Find files by pattern" }

func (GlobTool) Parameters() types.ToolDef {
	return types.ToolDef{
		Name:        "glob",
		Description: "Find files by pattern",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{"type": "string", "description": "Glob pattern"},
			},
			"required": []string{"pattern"},
		},
	}
}

func (GlobTool) Call(ctx context.Context, in Input) (*Result, error) {
	_ = ctx
	var args struct {
		Pattern string `json:"pattern"`
	}
	if err := json.Unmarshal([]byte(in.Arguments), &args); err != nil {
		return &Result{ID: in.ID, Output: err.Error(), Error: true}, nil
	}
	pattern := args.Pattern
	if !filepath.IsAbs(pattern) && in.CWD != "" {
		pattern = filepath.Join(in.CWD, pattern)
	}
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return &Result{ID: in.ID, Output: err.Error(), Error: true}, nil
	}
	sort.Strings(matches)
	return &Result{ID: in.ID, Output: strings.Join(matches, "\n")}, nil
}

type GitTool struct{}

func (GitTool) Name() string { return "git" }

func (GitTool) Description() string { return "Run git commands" }

func (GitTool) Parameters() types.ToolDef {
	return types.ToolDef{
		Name:        "git",
		Description: "Run git commands",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{"type": "string", "description": "Git command arguments"},
				"cwd":     map[string]any{"type": "string", "description": "Working directory"},
			},
			"required": []string{"command"},
		},
	}
}

func (GitTool) Call(ctx context.Context, in Input) (*Result, error) {
	var args struct {
		Command string `json:"command"`
		CWD     string `json:"cwd"`
	}
	if err := json.Unmarshal([]byte(in.Arguments), &args); err != nil {
		return &Result{ID: in.ID, Output: err.Error(), Error: true}, nil
	}
	parts := strings.Fields(args.Command)
	if len(parts) == 0 {
		return &Result{ID: in.ID, Output: "empty git command", Error: true}, nil
	}
	cmd := exec.CommandContext(ctx, "git", parts...)
	if args.CWD != "" {
		cmd.Dir = args.CWD
	} else if in.CWD != "" {
		cmd.Dir = in.CWD
	}
	out, err := cmd.CombinedOutput()
	return &Result{ID: in.ID, Output: string(out), Error: err != nil}, nil
}

type WebTool struct{}

func (WebTool) Name() string { return "web" }

func (WebTool) Description() string { return "Fetch a URL" }

func (WebTool) Parameters() types.ToolDef {
	return types.ToolDef{
		Name:        "web",
		Description: "Fetch a URL",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]any{"type": "string", "description": "URL to fetch"},
			},
			"required": []string{"url"},
		},
	}
}

func (WebTool) Call(ctx context.Context, in Input) (*Result, error) {
	var args struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(in.Arguments), &args); err != nil {
		return &Result{ID: in.ID, Output: err.Error(), Error: true}, nil
	}
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, args.URL, nil)
	if err != nil {
		return &Result{ID: in.ID, Output: err.Error(), Error: true}, nil
	}
	resp, err := client.Do(req)
	if err != nil {
		return &Result{ID: in.ID, Output: err.Error(), Error: true}, nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 20000))
	if err != nil {
		return &Result{ID: in.ID, Output: err.Error(), Error: true}, nil
	}
	return &Result{ID: in.ID, Output: string(body)}, nil
}

var tools = map[string]Tool{
	"bash": ShellTool{},
	"file": FileTool{},
	"grep": GrepTool{},
	"glob": GlobTool{},
	"git":  GitTool{},
	"web":  WebTool{},
}

func DefaultTools() map[string]Tool {
	return tools
}

func Definitions() []types.ToolDef {
	names := make([]string, 0, len(tools))
	for name := range tools {
		names = append(names, name)
	}
	sort.Strings(names)
	defs := make([]types.ToolDef, 0, len(names))
	for _, name := range names {
		defs = append(defs, tools[name].Parameters())
	}
	return defs
}

func ByName(name string) (Tool, bool) {
	t, ok := tools[name]
	return t, ok
}
