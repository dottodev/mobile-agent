package security

import "testing"

func TestDestructive(t *testing.T) {
	cases := []string{
		"rm -rf /",
		"rm -r build",
		"mkfs.ext4 /dev/sda",
		"dd if=/dev/zero of=/dev/sda",
	}
	for _, c := range cases {
		if !Destructive(c) {
			t.Fatalf("expected destructive: %s", c)
		}
	}
	if Destructive("ls -la") {
		t.Fatal("ls should not be destructive")
	}
}

func TestRules(t *testing.T) {
	cfg := &Config{
		ActiveProfile: "p",
		Profiles: map[string]Profile{
			"p": {
				Tools:    map[string]string{"bash": "ask", "file": "allow"},
				Commands: map[string]string{"rm -*": "deny"},
				Paths:    map[string]string{},
			},
		},
	}
	if action, _ := cfg.ToolAction("bash"); action != "ask" {
		t.Fatalf("unexpected tool action %q", action)
	}
	if action, _ := cfg.ToolAction("file"); action != "allow" {
		t.Fatalf("unexpected tool action %q", action)
	}
	if action, _ := cfg.CommandAction("rm -rf /"); action != "deny" {
		t.Fatalf("unexpected command action %q", action)
	}
}

func TestWildcardPrecedence(t *testing.T) {
	cfg := &Config{
		ActiveProfile: "p",
		Profiles: map[string]Profile{
			"p": {
				Commands: map[string]string{"*": "allow", "rm -*": "deny"},
			},
		},
	}
	if action, _ := cfg.CommandAction("rm -rf /"); action != "deny" {
		t.Fatalf("expected deny, got %q", action)
	}
	if action, _ := cfg.CommandAction("ls"); action != "allow" {
		t.Fatalf("expected allow, got %q", action)
	}
}
