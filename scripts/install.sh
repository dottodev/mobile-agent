#!/bin/sh
set -eu

GO=${GO:-go}
BIN_DIR="${HOME}/.local/bin"
CONFIG_DIR="${HOME}/.config/mobile-agent"
DATA_DIR="${HOME}/.local/share/mobile-agent"

mkdir -p "$BIN_DIR" "$CONFIG_DIR/prompts" "$DATA_DIR"

"$GO" build -o "$BIN_DIR/mobile-agent" ./cmd/mobile-agent

if [ -d prompts ]; then
  cp -r prompts/. "$CONFIG_DIR/prompts/"
fi

if [ ! -f "$CONFIG_DIR/config.json" ]; then
  cat > "$CONFIG_DIR/config.json" <<'JSON'
{
  "version": "1",
  "providers": {
    "openai": { "base_url": "https://api.openai.com/v1", "api_key": "", "api_key_env": "OPENAI_API_KEY", "model": "gpt-4o-mini" },
    "local": { "base_url": "http://127.0.0.1:11434/v1", "api_key": "", "api_key_env": "", "model": "local" }
  },
  "default_provider": "openai",
  "default_model": "gpt-4o-mini",
  "permission": {
    "default": "ask",
    "active_profile": "sandbox",
    "profiles": {
      "guarded": { "tools": { "bash": "ask", "file": "ask", "grep": "allow", "glob": "allow", "git": "ask", "web": "ask" }, "commands": {}, "paths": {} },
      "sandbox": { "tools": { "*": "allow" }, "commands": { "rm -*": "deny", "mkfs": "deny", "dd": "deny" }, "paths": {} },
      "unrestricted": { "tools": { "*": "allow" }, "commands": { "*": "allow" }, "paths": { "*": "allow" } }
    }
  }
}
JSON
  chmod 600 "$CONFIG_DIR/config.json"
fi

echo "installed $BIN_DIR/mobile-agent"
