package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type ProviderConfig struct {
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key"`
	APIKeyEnv string `json:"api_key_env"`
	Model     string `json:"model"`
}

type Profile struct {
	Tools    map[string]string `json:"tools"`
	Commands map[string]string `json:"commands"`
	Paths    map[string]string `json:"paths"`
}

type PermissionConfig struct {
	Default       string             `json:"default"`
	ActiveProfile string             `json:"active_profile"`
	Profiles      map[string]Profile `json:"profiles"`
}

type Config struct {
	Version         string                    `json:"version"`
	ConfigDir       string                    `json:"-"`
	DataDir         string                    `json:"-"`
	PromptsDir      string                    `json:"-"`
	Providers       map[string]ProviderConfig `json:"providers"`
	DefaultProvider string                    `json:"default_provider"`
	DefaultModel    string                    `json:"default_model"`
	Permission      PermissionConfig          `json:"permission"`
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return "/root"
}

func DefaultConfig() *Config {
	home := homeDir()
	return &Config{
		Version:         "1",
		ConfigDir:       filepath.Join(home, ".config", "mobile-agent"),
		DataDir:         filepath.Join(home, ".local", "share", "mobile-agent"),
		PromptsDir:      filepath.Join(home, ".config", "mobile-agent", "prompts"),
		DefaultProvider: "openai",
		DefaultModel:    "gpt-4o-mini",
		Providers: map[string]ProviderConfig{
			"openai": {
				BaseURL:   "https://api.openai.com/v1",
				APIKey:    "",
				APIKeyEnv: "OPENAI_API_KEY",
				Model:     "gpt-4o-mini",
			},
			"local": {
				BaseURL:   "http://127.0.0.1:11434/v1",
				APIKey:    "",
				APIKeyEnv: "",
				Model:     "local",
			},
		},
		Permission: PermissionConfig{
			Default:       "ask",
			ActiveProfile: "sandbox",
			Profiles: map[string]Profile{
				"guarded": {
					Tools:    map[string]string{"bash": "ask", "file": "ask", "grep": "allow", "glob": "allow", "git": "ask", "web": "ask"},
					Commands: map[string]string{},
					Paths:    map[string]string{},
				},
				"sandbox": {
					Tools:    map[string]string{"*": "allow"},
					Commands: map[string]string{"rm -*": "deny", "mkfs": "deny", "dd": "deny"},
					Paths:    map[string]string{},
				},
				"unrestricted": {
					Tools:    map[string]string{"*": "allow"},
					Commands: map[string]string{"*": "allow"},
					Paths:    map[string]string{"*": "allow"},
				},
			},
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := DefaultConfig()
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		if err == nil {
			if err := json.Unmarshal(data, cfg); err != nil {
				return nil, err
			}
		}
	}
	for name, p := range cfg.Providers {
		if p.APIKey == "" && p.APIKeyEnv != "" {
			p.APIKey = os.Getenv(p.APIKeyEnv)
			cfg.Providers[name] = p
		}
	}
	return cfg, nil
}

func (cfg *Config) Save(path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
