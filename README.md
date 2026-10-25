# mobile-agent

A local-first AI agent for Termux and other mobile Linux environments.

## Features

- Termux-friendly TUI with headless `--run` mode
- Bring-your-own-key providers through an OpenAI-compatible API
- Local model endpoints such as llama.cpp, Ollama, LM Studio, or any OpenAI-compatible server
- Named prompt stacks that can be listed, switched, and persisted per session
- Tools for shell, files, grep, glob, git, and web fetching
- Permission profiles: `guarded`, `sandbox`, and `unrestricted`
- Local session storage under `~/.local/share/mobile-agent`
- No telemetry and no network calls except to the configured provider and requested tools

## Build

```sh
go build -o mobile-agent ./cmd/mobile-agent
```

Android/Termux target:

```sh
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -buildmode=pie -o mobile-agent ./cmd/mobile-agent
```

## Install in Termux

```sh
pkg install golang git
git clone https://github.com/dottodev/mobile-agent
cd mobile-agent
sh scripts/install.sh
```

The install script builds the binary into `~/.local/bin/mobile-agent`, copies prompt stacks into `~/.config/mobile-agent/prompts`, and writes a default config to `~/.config/mobile-agent/config.json` with mode `0600`.

## Configure

Edit `~/.config/mobile-agent/config.json`:

```json
{
  "version": "1",
  "providers": {
    "openai": {
      "base_url": "https://api.openai.com/v1",
      "api_key": "",
      "api_key_env": "OPENAI_API_KEY",
      "model": "gpt-4o-mini"
    },
    "local": {
      "base_url": "http://127.0.0.1:11434/v1",
      "api_key": "",
      "api_key_env": "",
      "model": "local"
    }
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
```

Set `OPENAI_API_KEY` in the environment or place the key in `api_key`. For local servers that do not require a key, leave both empty.

Prompts and providers are selected through the CLI or config. Local models are supported by pointing a provider at an OpenAI-compatible endpoint.

## Usage

Interactive:

```sh
mobile-agent
```

One-shot:

```sh
mobile-agent --run "inspect this repository and summarize its build system"
```

Use a local model:

```sh
mobile-agent --provider local --model qwen3-coder
```

Auto-approve permission requests:

```sh
mobile-agent --auto
```

Start with a specific session or prompt stack:

```sh
mobile-agent --session <id> --stack review
```

## TUI commands

- `/help`
- `/session new`
- `/session list`
- `/session use <id>`
- `/prompt list`
- `/prompt use <name>`
- `/prompt show <name>`
- `/model <name>`
- `/provider <name>`
- `/perm guarded|sandbox|unrestricted`
- `/auto on|off`
- `/exit`

## Prompt stacks

Prompt stacks are JSON files under `~/.config/mobile-agent/prompts` or `./prompts` when running from a checkout.

```json
{
  "name": "build",
  "parts": [
    { "role": "system", "content": "You are a coding agent." }
  ]
}
```

A part can reference a file instead of inline content:

```json
{ "role": "system", "file": "/path/to/instructions.md" }
```

Stacks can be layered by adding more parts. The active stack is stored with the session.

## Notes

- Session data is local JSON under `~/.local/share/mobile-agent`.
- The agent never binds a network listener in this version.
- Permissions are enforced before tool execution.
- Destructive shell commands require confirmation unless the active profile or `--auto` allows them.
