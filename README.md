# nsh — Natural Shell

`nsh` is a natural-language shell that translates plain English into real, executable shell commands. It can also answer questions, inspect your system, use built-in tools (search, git, web), and execute commands with safety controls.

**Local-first by default:** `nsh` uses **LM Studio** (local, private, free) as the default LLM provider, with **Google Gemini** as an optional cloud alternative.

---

## Why nsh?

- Type what you want: `nsh "find large files in this folder"`
- Get a safe command + explanation
- **Auto-run safe commands**, **confirm risky ones**
- REPL mode for interactive usage: `nsh`

---

## Features

### Core
- **Natural language → real shell commands** (not internal tool names)
- **REPL mode** with quick commands (`:help`, `:history`, etc.)
- **Built-in tools** for system introspection, file search, web search, and git

### Safety & Execution
- **Risk classification**: Low / Medium / High / Blocked
- **Install command detection**: `brew install`, `apt install`, `pip install`, `npm install`, etc. always require confirmation
- **LLM-driven execution policy**:
  - The model returns an execution policy: `auto` or `confirm`
  - `nsh` will **only escalate** to confirmation (it will not bypass hard-coded safety rules)
- **Confidence threshold for auto-execution**:
  - If the model's `confidence` is below `MinAutoExecConfidence`, `nsh` will require confirmation

### Provider & Configuration
- **LM Studio is default provider** (local, private)
- **Gemini as cloud alternative**
- **Universal configuration via environment variables**
- **Config file support**: `~/.config/nsh/config.yml` (or `$XDG_CONFIG_HOME/nsh/config.yml`)
- **Config priority**: **env vars > config file > defaults**

### UX & Robustness
- **Markdown rendering in terminal output** (via `glamour`)
- **Robust JSON parsing** for imperfect LLM outputs:
  - Extracts first valid JSON object/array
  - Repairs common issues (trailing commas, control chars in strings, code fences, token noise)

### Workspace File Index (REPL)
Fast "does this file exist / where is it" workflows using a workspace index:

| Command | Description |
|---------|-------------|
| `:index` / `:files` | Index summary |
| `:refresh` | Rebuild/refresh index |
| `:exists <path>` | Check existence (e.g. `:exists package.json`) |
| `:where <name>` | Find by filename (e.g. `:where auth.go`) |
| `:ext <ext>` | List by extension (e.g. `:ext go`) |

---

## Architecture

`nsh` is organized as a small pipeline: collect runtime context, ask the selected LLM for a structured response, run safety checks, then either answer, ask for confirmation, or execute the generated shell command.

```text
User input
  -> Context collection (`internal/context`)
  -> Provider selection + prompt generation (`internal/llm`, `internal/config`)
  -> Structured LLM output
     - mode: `answer` | `command` | `plan` | `clarify`
     - execution: `auto` | `confirm`
  -> Safety evaluation (`internal/safety`)
  -> Execution or follow-up
     - answer / clarify shown in terminal UI (`internal/ui`)
     - command executed via shell (`internal/executor`)
     - tool-backed plans use built-ins from (`internal/tools`)
  -> History persisted (`internal/history`)
```

### Request Flow

1. `internal/context` captures the current working directory, shell, OS/arch, git metadata, detected project type, and optional workspace file index.
2. `internal/config` loads defaults, then merges `~/.config/nsh/config.yml`, then applies environment variable overrides.
3. `internal/llm` sends the prompt to the active provider:
   - `LMStudioProvider` for local OpenAI-compatible models
   - `GeminiProvider` for Google Gemini
4. The model returns structured JSON, parsed into a `Generated` response with a mode, command/message, confidence, risk hints, and execution policy.
5. `internal/safety` applies hard blocks, regex-based risk detection, install-command detection, and confirmation rules from config.
6. If execution is allowed, `internal/executor` runs the command in the user’s shell and current working directory, captures stdout/stderr, and returns the exit code.
7. `internal/ui` renders answers, plans, commands, confirmations, and command output in the terminal.
8. `internal/history` stores recent interactions in `~/.config/nsh/history.json`.

### Internal Packages

| Package | Responsibility |
|---------|----------------|
| `internal/config` | Defaults, YAML config loading, env var overrides, provider settings |
| `internal/context` | Runtime environment discovery, git/project detection, file index lifecycle |
| `internal/llm` | Provider abstraction, prompt construction, JSON cleanup/parsing, structured response model |
| `internal/safety` | Risk classification, blocked/high/medium checks, install-command confirmation rules |
| `internal/executor` | Shell command execution, output capture, exit-code reporting |
| `internal/tools` | Built-in helpers for file search, content search, diagnostics, ports, processes, web/package helpers |
| `internal/fileindex` | Workspace index build/query/refresh/watch logic for fast REPL file lookups |
| `internal/ui` | Terminal rendering, markdown output, prompts, spinners, styled command/result views |
| `internal/history` | Persistent local history for recent commands and outcomes |

### File Index Subsystem

The workspace index is a dedicated subsystem used for fast REPL file queries such as `:exists`, `:where`, and `:ext`.

- In git repos, `internal/fileindex` prefers `git ls-files` plus untracked files for fast, relevant indexing.
- Outside git repos, it falls back to a filesystem walk.
- The index stores relative paths, names, extensions, file sizes, modification times, and inferred language metadata.
- It refreshes when stale, when `HEAD` changes, or when explicitly rebuilt.
- `fsnotify`-based watching keeps the index updated during REPL sessions.

### Design Notes

- Local-first by default: LM Studio is the default provider and keeps the common path offline/private.
- Structured generation: providers return JSON instead of free-form text, which makes safety and execution decisions deterministic.
- Safety before execution: LLM output never bypasses hardcoded blocked patterns or confirmation rules.
- Terminal-native UX: rendering and execution remain shell-centric rather than abstracting commands into opaque actions.

---

## Installation

### Option A: From Source (Recommended)

```bash
git clone https://github.com/riddhishganeshmahajan/nsh.git
cd nsh

make install
# or:
go build -o nsh ./cmd/nsh
sudo cp nsh /usr/local/bin/
```

### Option B: Go Install

```bash
go install github.com/riddhishganeshmahajan/nsh/cmd/nsh@latest
```

Verify:

```bash
nsh --version
```

---

## Provider Setup

### Default: LM Studio (Local, Private)

1. Install LM Studio: https://lmstudio.ai  
2. Start LM Studio and **load a model**
3. Ensure the server is running (OpenAI-compatible endpoint)

Default expected URL: `http://localhost:1234/v1`

You can then run:

```bash
nsh "list files"
```

For guided setup (recommended):

```bash
nsh --setup
```

#### LM Studio Environment Variables

```bash
export NSH_PROVIDER=lmstudio
export NSH_LMSTUDIO_URL="http://localhost:1234/v1"
export NSH_LMSTUDIO_MODEL="local-model"
export NSH_LMSTUDIO_TIMEOUT="120"
```

> **Tip:** `NSH_LMSTUDIO_MODEL` should match the model identifier LM Studio exposes. If your LM Studio server ignores `model`, it will typically use the loaded model anyway.

---

### Alternative: Google Gemini (Cloud)

1. Get an API key for Gemini
2. Export the key (default env var name is `GEMINI_API_KEY`)
3. Set provider to `gemini`

```bash
export GEMINI_API_KEY="your-api-key"
export NSH_PROVIDER=gemini
```

Run:

```bash
nsh "what is kubernetes"
nsh "find all go files"
```

Optional Gemini env overrides:

```bash
export NSH_GEMINI_MODEL="gemini-2.5-flash"
export NSH_GEMINI_TIMEOUT="30"
```

---

## Usage

### One-Shot Mode

```bash
nsh "show disk usage"
nsh "find all python files under src"
nsh "what's using port 8080"
nsh "search the web for golang error handling best practices"
```

### REPL Mode

```bash
nsh
```

Inside the REPL, type naturally or use quick commands:

| Command | Description |
|---------|-------------|
| `:help` | Show help |
| `:history` / `:h` | Command history |
| `:again` / `:redo` | Repeat last command |
| `:context` / `:ctx` | Show current context |
| `:diag` | System diagnostics |
| `:git` | Git status |
| `:search <query>` | Web search |
| `:find <pattern>` | Find files |
| `:grep <pattern>` | Search file contents |
| `:port <num>` | Check port usage |
| `:ps <name>` | Find processes |
| `:pkg <name>` | Search packages |
| `:dry` | Toggle dry run mode |
| `:learn` | Toggle learn mode |
| `!<cmd>` | Run raw shell command |
| `exit` / `quit` | Exit nsh |

### Flags

| Flag | Description |
|------|-------------|
| `--dry` | Dry-run mode (don't execute commands) |
| `--learn` | Show extra explanations / alternatives |
| `--setup` | Interactive setup wizard |
| `--force` | Override blocked commands (**dangerous**) |
| `--version` | Print version |

Examples:

```bash
nsh --dry "delete all log files"
nsh --learn "find which process is on port 3000"
nsh --setup
```

---

## Execution Policy & Safety

When `nsh` generates a command, it decides whether to run it automatically or require confirmation using three inputs:

1. **Hardcoded safety rules** (blocked/high/medium patterns)
2. **LLM execution policy** (`execution: auto|confirm`)
3. **Confidence threshold** (`MinAutoExecConfidence`)

### What Auto-Executes

`nsh` will auto-execute only when it is confident the command is safe and read-only:

- `ls -la`
- `find . -name '*.go'`
- `ps aux | grep ...`
- `df -h`
- `cat`, `head`, `tail`, `wc`

### What Always Requires Confirmation

Even if the model suggests auto, `nsh` escalates to confirmation for:

- File modification / deletion (`rm`, `mv`, `cp`)
- `sudo` commands
- Destructive git operations (`reset --hard`, `clean -fdx`, force push)
- **Software installs** (detected automatically):
  - `brew install ...`
  - `apt install ...` / `apt-get install ...`
  - `pip install ...` / `pip3 install ...`
  - `npm install ...` / `yarn add ...` / `pnpm add ...`
  - `cargo install ...`
  - `go install ...`
  - `gem install ...`
  - And many more package managers

### Risk Levels

| Level | Behavior | Examples |
|-------|----------|----------|
| **Low** | Auto-execute (if LLM says `auto`) | `ls`, `cat`, `git status` |
| **Medium** | Confirm by default | `rm file`, `curl`, `git push` |
| **High** | Explicit confirm | `rm -rf`, `git reset --hard`, `sudo` |
| **Blocked** | Never execute | `rm -rf /`, fork bombs, pipe to shell |

---

## Configuration

### Configuration Priority

1. **Environment variables** (highest priority)
2. **Config file**: `~/.config/nsh/config.yml`
3. **Built-in defaults** (lowest priority)

### Environment Variables

| Variable | Purpose | Example |
|----------|---------|---------|
| `NSH_PROVIDER` | LLM provider: `lmstudio` or `gemini` | `lmstudio` |
| `NSH_LMSTUDIO_URL` | LM Studio base URL | `http://localhost:1234/v1` |
| `NSH_LMSTUDIO_MODEL` | LM Studio model name/id | `deepseek/deepseek-r1-0528-qwen3-8b` |
| `NSH_LMSTUDIO_TIMEOUT` | LM Studio timeout seconds | `120` |
| `NSH_GEMINI_MODEL` | Gemini model name | `gemini-2.5-flash` |
| `NSH_GEMINI_TIMEOUT` | Gemini timeout seconds | `30` |
| `GEMINI_API_KEY` | Gemini API key (default env var) | `...` |

### Full Config File Example

Create `~/.config/nsh/config.yml`:

```yaml
# LLM provider: "lmstudio" (default) or "gemini"
provider: lmstudio

lmstudio:
  # OpenAI-compatible base URL exposed by LM Studio
  base_url: "http://localhost:1234/v1"
  # Model identifier (often the currently loaded model in LM Studio)
  model: "local-model"
  timeout_seconds: 120

gemini:
  # nsh reads the API key from this env var (default: GEMINI_API_KEY)
  api_key_env: "GEMINI_API_KEY"
  model: "gemini-2.5-flash"
  timeout_seconds: 30

ui:
  # Safety confirmations
  always_confirm: false
  confirm_medium: true
  confirm_high: true

  # Output preferences
  color: true
  learn_mode: false

  # If model confidence is below this, nsh will ask for confirmation
  # even for otherwise low-risk commands (0.0 - 1.0)
  min_auto_exec_confidence: 0.8

exec:
  dry_run: false
  use_login_shell: false

history:
  max_entries: 50

safety:
  # Hard blocks (never run unless --force)
  block_patterns:
    - 'rm\s+-rf\s+/\s*$'
    - 'rm\s+-rf\s+~'
    - 'rm\s+-rf\s+\$HOME'
    - ':\(\)\{\s*:\|:&\s*\};:'
    - 'mkfs\.'
    - 'dd\s+if=.*/dev/'
    - 'curl.*\|\s*(sh|bash|zsh)'
    - 'wget.*\|\s*(sh|bash|zsh)'

  # High-risk patterns (confirmation required)
  high_patterns:
    - '\brm\b.*-rf\b'
    - '\bgit\b.*reset\s+--hard'
    - '\bgit\b.*clean\s+-fdx'
    - '\bgit\b.*push.*--force'
    - '\bchmod\b.*-R'
    - '\bchown\b.*-R'
    - '\bsudo\b'

  # Medium-risk patterns (confirmation recommended)
  medium_patterns:
    - '\bcurl\b'
    - '\bwget\b'
    - '\bssh\b'
    - '\bscp\b'
    - '\brsync\b'
    - '\brm\b'
    - '\bgit\b.*push'
    - '\bgit\b.*commit'
    - '\bgit\b.*rebase'
```

---

## Examples

### Natural Language Queries

```bash
# Ask questions
nsh "what is kubernetes"
nsh "explain the difference between docker and podman"

# System queries
nsh "is docker running"
nsh "what's using port 8080"
nsh "show me memory usage"
nsh "show system diagnostics"

# File operations
nsh "find all javascript files"
nsh "search for TODO comments in src"
nsh "list files in current directory"

# Git operations
nsh "show recent commits"
nsh "what files have changed"

# Web search
nsh "search the web for golang tutorials"

# Commands
nsh "compress this folder as backup"
nsh "kill process on port 3000"
```

### LLM-Driven Execution Policy

```bash
nsh "delete all *.log files"
```

Expected behavior:
- `nsh` shows the generated command (e.g. `rm *.log`)
- Requires confirmation (LLM sets `execution: "confirm"`)

### Install Command Detection

```bash
nsh "install ripgrep"
```

If `nsh` generates `brew install ripgrep` or `apt install ripgrep`, it will **always require confirmation** even if the model suggests auto-execution.

### Environment Variable Overrides

Temporarily switch provider without editing config:

```bash
# Use Gemini
NSH_PROVIDER=gemini GEMINI_API_KEY="..." nsh "explain docker compose"

# Use LM Studio with specific model
NSH_LMSTUDIO_MODEL="deepseek/deepseek-r1" nsh "show disk usage"
```

### File Index Commands (REPL)

```bash
nsh
> :index          # Show workspace file summary
> :exists package.json
> :where auth.go
> :ext go
> :refresh        # Rebuild index
```

---

## Troubleshooting

### LM Studio Errors

- Ensure LM Studio is running and a model is loaded
- Confirm the server URL (default `http://localhost:1234/v1`)
- Check which port LM Studio is using in its settings
- Try increasing timeout: `NSH_LMSTUDIO_TIMEOUT=180`

### Gemini Errors

- Ensure your API key env var is set: `export GEMINI_API_KEY="..."`
- Confirm provider: `export NSH_PROVIDER=gemini`

### "Invalid Command: Internal Tool Name"

`nsh` refuses to execute internal tool names as shell commands. Rephrase your request (e.g. "list files" instead of asking for a tool directly).

### JSON Parsing Errors

If you see JSON parsing errors with certain models, the robust parser should handle most cases. If issues persist, try a different model or report the issue.

---

## Security Notes

- `nsh` is designed to be conservative:
  - Blocked patterns are never executed unless you pass `--force`
  - Installs always require confirmation
  - Low confidence escalates to confirmation
- **Always review commands before confirming**, especially anything involving `sudo`, deletes, or network operations.

---

