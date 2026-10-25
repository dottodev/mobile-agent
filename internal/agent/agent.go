package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"mobile-agent/internal/config"
	"mobile-agent/internal/prompt"
	"mobile-agent/internal/provider"
	"mobile-agent/internal/security"
	"mobile-agent/internal/store"
	"mobile-agent/internal/tool"
	"mobile-agent/internal/types"
)

type Prompter interface {
	Confirm(prompt string) (bool, error)
}

type Agent struct {
	store           *store.Store
	provider        provider.Provider
	checker         *security.Config
	tools           map[string]tool.Tool
	prompter        Prompter
	out             io.Writer
	auto            bool
	session         *types.Session
	stack           *prompt.Stack
	maxSteps        int
	promptsDir      string
	defaultProvider string
	defaultModel    string
}

func New(st *store.Store, prov provider.Provider, cfg *config.Config, prompter Prompter, out io.Writer, auto bool) (*Agent, error) {
	promptsDir := cfg.PromptsDir
	if _, err := os.Stat(promptsDir); err != nil {
		if _, err := os.Stat("prompts"); err == nil {
			promptsDir = "prompts"
		}
	}
	stack, err := prompt.LoadStack(promptsDir, "build")
	if err != nil {
		stack = prompt.BuiltIn["build"]
	}

	checker := &security.Config{
		Default:       cfg.Permission.Default,
		ActiveProfile: cfg.Permission.ActiveProfile,
		Profiles:      map[string]security.Profile{},
		Auto:          auto,
	}
	for name, p := range cfg.Permission.Profiles {
		checker.Profiles[name] = security.Profile{
			Tools:    p.Tools,
			Commands: p.Commands,
			Paths:    p.Paths,
		}
	}
	if _, ok := checker.Profiles[checker.ActiveProfile]; !ok {
		checker.ActiveProfile = "guarded"
	}

	return &Agent{
		store:           st,
		provider:        prov,
		checker:         checker,
		tools:           tool.DefaultTools(),
		prompter:        prompter,
		out:             out,
		auto:            auto,
		stack:           stack,
		maxSteps:        25,
		promptsDir:      promptsDir,
		defaultProvider: cfg.DefaultProvider,
		defaultModel:    cfg.DefaultModel,
	}, nil
}

func (a *Agent) NewSession(providerName, model string) error {
	if providerName == "" {
		providerName = a.defaultProvider
	}
	if model == "" {
		model = a.defaultModel
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	stackName := "build"
	if a.stack != nil {
		stackName = a.stack.Name
	}
	a.session = store.NewSession(store.NewID(), providerName, model, stackName, cwd)
	return a.store.Save(a.session)
}

func (a *Agent) LoadSession(id string) error {
	sess, err := a.store.Load(id)
	if err != nil {
		return err
	}
	a.session = sess
	if sess.Stack != "" {
		if stack, err := prompt.LoadStack(a.promptsDir, sess.Stack); err == nil {
			a.stack = stack
		}
	}
	return nil
}

func (a *Agent) SetSession(sess *types.Session) {
	a.session = sess
}

func (a *Agent) Session() *types.Session {
	return a.session
}

func (a *Agent) SetStack(name string) error {
	stack, err := prompt.LoadStack(a.promptsDir, name)
	if err != nil {
		return err
	}
	a.stack = stack
	if a.session != nil {
		a.session.Stack = name
	}
	return nil
}

func (a *Agent) SetModel(model string) {
	if a.session != nil {
		a.session.Model = model
	}
}

func (a *Agent) SetProfile(profile string) {
	a.checker.ActiveProfile = profile
}

func (a *Agent) SetProvider(prov provider.Provider) {
	a.provider = prov
}

func (a *Agent) SetAuto(auto bool) {
	a.auto = auto
	a.checker.Auto = auto
}

func (a *Agent) Run(ctx context.Context, user string) error {
	if a.session == nil {
		if err := a.NewSession("", ""); err != nil {
			return err
		}
	}

	a.session.Messages = append(a.session.Messages, types.Message{Role: "user", Content: user})
	if a.session.Title == "" {
		a.session.Title = truncate(user, 60)
	}

	for step := 0; step < a.maxSteps; step++ {
		req := types.CompletionRequest{
			Model:       a.session.Model,
			Messages:    a.composeMessages(),
			Tools:       tool.Definitions(),
			Temperature: 0.2,
		}
		resp, err := a.provider.Complete(ctx, req)
		if err != nil {
			return err
		}

		a.session.Messages = append(a.session.Messages, resp.Message)

		if len(resp.Message.ToolCalls) == 0 {
			if resp.Message.Content != "" {
				fmt.Fprintln(a.out, resp.Message.Content)
			}
			return a.store.Save(a.session)
		}

		for _, tc := range resp.Message.ToolCalls {
			res := a.execute(ctx, tc)
			if len(res.Output) > 8000 {
				res.Output = res.Output[:8000] + "\n...[truncated]..."
			}
			a.session.Messages = append(a.session.Messages, types.Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    res.Output,
			})
			fmt.Fprintf(a.out, "[tool %s]\n%s\n", tc.Name, res.Output)
		}

		if err := a.store.Save(a.session); err != nil {
			return err
		}
	}

	return errors.New("max steps exceeded")
}

func (a *Agent) composeMessages() []types.Message {
	msgs := []types.Message{}
	if a.stack != nil {
		msgs = append(msgs, prompt.Compose(a.stack, "")...)
	}
	if a.session != nil {
		msgs = append(msgs, a.session.Messages...)
	}
	return msgs
}

func (a *Agent) execute(ctx context.Context, tc types.ToolCall) tool.Result {
	t, ok := a.tools[tc.Name]
	if !ok {
		return tool.Result{ID: tc.ID, Output: "unknown tool: " + tc.Name, Error: true}
	}

	var args struct {
		Command string `json:"command"`
		CWD     string `json:"cwd"`
		Path    string `json:"path"`
		Op      string `json:"op"`
	}
	if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
		return tool.Result{ID: tc.ID, Output: err.Error(), Error: true}
	}

	in := tool.Input{
		ID:        tc.ID,
		Name:      tc.Name,
		Arguments: tc.Arguments,
		SessionID: a.session.ID,
		CWD:       a.session.CWD,
	}
	if args.CWD != "" {
		in.CWD = args.CWD
	}

	var allowed bool
	var err error
	switch tc.Name {
	case "bash":
		allowed, err = a.permitShell(args.Command)
	case "file":
		allowed, err = a.permitPath(args.Op, args.Path)
	default:
		allowed, err = a.permitTool(tc.Name)
	}
	if err != nil {
		return tool.Result{ID: tc.ID, Output: err.Error(), Error: true}
	}
	if !allowed {
		return tool.Result{ID: tc.ID, Output: "permission denied", Error: true}
	}

	res, err := t.Call(ctx, in)
	if err != nil {
		return tool.Result{ID: tc.ID, Output: err.Error(), Error: true}
	}
	return tool.Result{ID: res.ID, Output: res.Output, Error: res.Error}
}

func (a *Agent) permitTool(name string) (bool, error) {
	action, _ := a.checker.ToolAction(name)
	switch action {
	case "allow":
		return true, nil
	case "deny":
		return false, nil
	}
	if a.auto {
		return true, nil
	}
	return a.prompter.Confirm("Allow tool: " + name)
}

func (a *Agent) permitShell(cmd string) (bool, error) {
	action, _ := a.checker.CommandAction(cmd)
	if action == "deny" {
		return false, nil
	}
	if security.Destructive(cmd) {
		if action == "allow" || a.auto {
			return true, nil
		}
		return a.prompter.Confirm("Allow destructive command: " + cmd)
	}
	return a.permitTool("bash")
}

func (a *Agent) permitPath(op, path string) (bool, error) {
	if path == "" {
		return true, nil
	}
	action, _ := a.checker.PathAction(path)
	switch action {
	case "allow":
		return true, nil
	case "deny":
		return false, nil
	}
	if a.auto {
		return true, nil
	}
	return a.prompter.Confirm(fmt.Sprintf("Allow file %s: %s", op, path))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
