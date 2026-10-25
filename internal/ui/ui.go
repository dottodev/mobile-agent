package ui

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"mobile-agent/internal/agent"
	"mobile-agent/internal/config"
	"mobile-agent/internal/prompt"
	"mobile-agent/internal/provider"
	"mobile-agent/internal/store"
)

var errExit = errors.New("exit")

type CLI struct {
	reader *bufio.Reader
	out    io.Writer
}

func (c *CLI) Confirm(text string) (bool, error) {
	fmt.Fprintf(c.out, "%s [y/N/a] ", text)
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes", "a", "always":
		return true, nil
	default:
		return false, nil
	}
}

func Run(args []string) error {
	fs := flag.NewFlagSet("mobile-agent", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	runPrompt := fs.String("run", "", "run one prompt and exit")
	sessionID := fs.String("session", "", "session id")
	stackName := fs.String("stack", "", "prompt stack")
	model := fs.String("model", "", "model")
	providerName := fs.String("provider", "", "provider")
	auto := fs.Bool("auto", false, "auto approve permission requests")
	configPath := fs.String("config", "", "config path")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	cfgDefault := config.DefaultConfig()
	cfgFile := *configPath
	if cfgFile == "" {
		cfgFile = filepath.Join(cfgDefault.ConfigDir, "config.json")
	}
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return err
	}
	if *providerName != "" {
		cfg.DefaultProvider = *providerName
	}
	if *model != "" {
		cfg.DefaultModel = *model
	}

	st := store.New(cfg.DataDir)
	prov, err := provider.New(cfg, cfg.DefaultProvider, cfg.DefaultModel)
	if err != nil {
		return err
	}

	cli := &CLI{reader: bufio.NewReader(os.Stdin), out: os.Stdout}
	a, err := agent.New(st, prov, cfg, cli, cli.out, *auto)
	if err != nil {
		return err
	}

	if *stackName != "" {
		if err := a.SetStack(*stackName); err != nil {
			return err
		}
	}
	if *sessionID != "" {
		if err := a.LoadSession(*sessionID); err != nil {
			return err
		}
	}
	if *runPrompt != "" {
		return a.Run(context.Background(), *runPrompt)
	}

	return cli.loop(a, cfg, st, cfgFile)
}

func (c *CLI) loop(a *agent.Agent, cfg *config.Config, st *store.Store, cfgFile string) error {
	fmt.Fprintln(c.out, "mobile-agent: type /help for commands")
	for {
		fmt.Fprint(c.out, "\033[1;32m>\033[0m ")
		line, err := c.reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "/") {
			if err := c.command(line, a, cfg, st, cfgFile); err != nil {
				if errors.Is(err, errExit) {
					return nil
				}
				fmt.Fprintln(c.out, "error:", err)
			}
			continue
		}
		if err := a.Run(context.Background(), line); err != nil {
			fmt.Fprintln(c.out, "error:", err)
		}
	}
}

func (c *CLI) command(line string, a *agent.Agent, cfg *config.Config, st *store.Store, cfgFile string) error {
	parts := strings.Fields(line)
	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "/help":
		fmt.Fprintln(c.out, "/help /exit /session new|list|use <id> /prompt list|use|show <name> /model <name> /provider <name> /perm <profile> /auto on|off")
		return nil
	case "/exit", "/quit":
		return errExit
	case "/session":
		return c.sessionCommand(args, a, st)
	case "/prompt", "/stack":
		return c.promptCommand(args, a, cfg)
	case "/model":
		if len(args) == 0 {
			if a.Session() != nil {
				fmt.Fprintln(c.out, a.Session().Model)
			}
			return nil
		}
		a.SetModel(args[0])
		if a.Session() != nil {
			_ = st.Save(a.Session())
		}
		fmt.Fprintln(c.out, "model:", args[0])
		return nil
	case "/provider":
		if len(args) == 0 {
			fmt.Fprintln(c.out, cfg.DefaultProvider)
			return nil
		}
		model := cfg.DefaultModel
		if a.Session() != nil && a.Session().Model != "" {
			model = a.Session().Model
		}
		prov, err := provider.New(cfg, args[0], model)
		if err != nil {
			return err
		}
		a.SetProvider(prov)
		cfg.DefaultProvider = args[0]
		if a.Session() != nil {
			a.Session().Provider = args[0]
			_ = st.Save(a.Session())
		}
		_ = cfg.Save(cfgFile)
		fmt.Fprintln(c.out, "provider:", args[0])
		return nil
	case "/perm":
		if len(args) == 0 {
			fmt.Fprintln(c.out, cfg.Permission.ActiveProfile)
			return nil
		}
		a.SetProfile(args[0])
		cfg.Permission.ActiveProfile = args[0]
		_ = cfg.Save(cfgFile)
		fmt.Fprintln(c.out, "profile:", args[0])
		return nil
	case "/auto":
		if len(args) == 0 {
			fmt.Fprintln(c.out, "auto")
			return nil
		}
		on := args[0] == "on" || args[0] == "true" || args[0] == "1"
		a.SetAuto(on)
		fmt.Fprintln(c.out, "auto:", on)
		return nil
	default:
		fmt.Fprintln(c.out, "unknown command:", cmd)
		return nil
	}
}

func (c *CLI) sessionCommand(args []string, a *agent.Agent, st *store.Store) error {
	if len(args) == 0 {
		fmt.Fprintln(c.out, "usage: /session new|list|use <id>")
		return nil
	}
	switch args[0] {
	case "new":
		if err := a.NewSession("", ""); err != nil {
			return err
		}
		fmt.Fprintln(c.out, "session:", a.Session().ID)
	case "list":
		sessions, err := st.List()
		if err != nil {
			return err
		}
		for _, s := range sessions {
			fmt.Fprintf(c.out, "%s  %s  %s\n", s.ID, s.Updated.Format("2006-01-02 15:04"), s.Title)
		}
	case "use":
		if len(args) < 2 {
			return errors.New("missing session id")
		}
		if err := a.LoadSession(args[1]); err != nil {
			return err
		}
		fmt.Fprintln(c.out, "session:", args[1])
	default:
		fmt.Fprintln(c.out, "usage: /session new|list|use <id>")
	}
	return nil
}

func (c *CLI) promptCommand(args []string, a *agent.Agent, cfg *config.Config) error {
	if len(args) == 0 {
		fmt.Fprintln(c.out, "usage: /prompt list|use|show <name>")
		return nil
	}
	switch args[0] {
	case "list":
		names, err := prompt.List(cfg.PromptsDir)
		if err != nil {
			return err
		}
		fmt.Fprintln(c.out, strings.Join(names, "\n"))
	case "use":
		if len(args) < 2 {
			return errors.New("missing prompt name")
		}
		if err := a.SetStack(args[1]); err != nil {
			return err
		}
		fmt.Fprintln(c.out, "prompt:", args[1])
	case "show":
		if len(args) < 2 {
			return errors.New("missing prompt name")
		}
		stack, err := prompt.LoadStack(cfg.PromptsDir, args[1])
		if err != nil {
			return err
		}
		for _, part := range stack.Parts {
			fmt.Fprintf(c.out, "[%s] %s\n", part.Role, part.Content)
		}
	default:
		fmt.Fprintln(c.out, "usage: /prompt list|use|show <name>")
	}
	return nil
}
