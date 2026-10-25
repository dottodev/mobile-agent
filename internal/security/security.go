package security

import (
	"regexp"
	"strings"
)

type Config struct {
	Default       string
	ActiveProfile string
	Profiles      map[string]Profile
	Auto          bool
}

type Profile struct {
	Tools    map[string]string
	Commands map[string]string
	Paths    map[string]string
}

func (c *Config) profile() Profile {
	return c.Profiles[c.ActiveProfile]
}

func matchRule(rules map[string]string, key string) (string, bool) {
	if rules == nil {
		return "", false
	}
	if v, ok := rules[key]; ok {
		return v, true
	}
	for pat, act := range rules {
		if pat == "*" {
			continue
		}
		if strings.HasSuffix(pat, "*") && strings.HasPrefix(key, strings.TrimSuffix(pat, "*")) {
			return act, true
		}
	}
	if v, ok := rules["*"]; ok {
		return v, true
	}
	return "", false
}

func (c *Config) ToolAction(name string) (string, bool) {
	return matchRule(c.profile().Tools, name)
}

func (c *Config) CommandAction(cmd string) (string, bool) {
	return matchRule(c.profile().Commands, cmd)
}

func (c *Config) PathAction(path string) (string, bool) {
	return matchRule(c.profile().Paths, path)
}

var destructive = regexp.MustCompile(`(?i)(\brm\s+-r|\bmkfs|\bdd\s|:\(\)\s*\{|\bchmod\s+-R|\bchown\s+-R|>\s*/dev/(null|zero|mem|kmem))`)

func Destructive(cmd string) bool {
	return destructive.MatchString(cmd)
}
