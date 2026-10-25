#!/bin/sh
set -eu

GO=${GO:-go}
PREFIX=${PREFIX:-}
HOME=${HOME:?}
CONFIG_DIR="$HOME/.config/mobile-agent"
DATA_DIR="$HOME/.local/share/mobile-agent"

if ! command -v "$GO" >/dev/null 2>&1; then
  echo "error: '$GO' not found in PATH." >&2
  echo "install Go first:  pkg install golang" >&2
  exit 1
fi

if [ -n "$PREFIX" ] && [ -d "$PREFIX/bin" ] && [ -w "$PREFIX/bin" ]; then
  BIN_DIR="$PREFIX/bin"
elif [ -d "$HOME/bin" ] || [ -w "$HOME" ]; then
  BIN_DIR="$HOME/bin"
else
  BIN_DIR="$HOME/.local/bin"
fi

mkdir -p "$BIN_DIR" "$CONFIG_DIR/prompts" "$DATA_DIR"

"$GO" build -o "$BIN_DIR/mobile-agent" ./cmd/mobile-agent
chmod 755 "$BIN_DIR/mobile-agent"

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

case ":${PATH:-}:" in
  *":$BIN_DIR:"*) IN_PATH=yes ;;
  *) IN_PATH=no ;;
esac

if [ "$IN_PATH" = no ]; then
  for RC in "$HOME/.bashrc" "$HOME/.profile"; do
    if [ ! -f "$RC" ]; then
      : > "$RC"
    fi
    if ! grep -Fq "export PATH=\"$BIN_DIR:\$PATH\"" "$RC"; then
      printf '\nexport PATH="%s:$PATH"\n' "$BIN_DIR" >> "$RC"
    fi
  done
fi

echo "installed $BIN_DIR/mobile-agent"
echo "config:   $CONFIG_DIR/config.json"
echo "sessions: $DATA_DIR"

if [ "$IN_PATH" = no ]; then
  echo
  echo "$BIN_DIR is not in PATH. Activate it now with:"
  echo "  export PATH=\"$BIN_DIR:\$PATH\""
  echo "and make it permanent:"
  echo "  echo 'export PATH=\"$BIN_DIR:\$PATH\"' >> ~/.bashrc"
  echo "  source ~/.bashrc"
  echo "then run: mobile-agent --help"
else
  echo "run: mobile-agent --help"
fi
