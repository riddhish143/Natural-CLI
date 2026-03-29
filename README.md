# nsh — Natural Shell
<img width="1352" height="508" alt="image" src="https://github.com/user-attachments/assets/9ff8d7fb-d6c1-4a89-b077-313972c1ad32" />


`nsh` is a natural-language shell that translates plain English into real, executable shell commands. It can also answer questions, inspect your system, use built-in tools (search, git, web), and execute commands with safety controls.

**Local-first by default:** `nsh` uses **LM Studio** (local, private, free) as the default LLM provider, with **Google Gemini** as an optional cloud alternative.

---

## Why nsh?

- Type what you want: `nsh "find large files in this folder"`
- Get a safe command + explanation
- **Auto-run safe commands**, **confirm risky ones**
- REPL mode for interactive usage: `nsh`

---

## Architecture

`nsh` is built with a modular architecture that separates concerns and enables extensibility:

```
┌─────────────────────────────────────────────────────────────┐
│                         User Input                          │
│                    (Natural Language)                       │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                      UI Layer                               │
│  • Terminal Interface (Markdown rendering via glamour)      │
│  • REPL Mode (interactive commands)                         │
│  • Animations & Styling                                     │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                    Context Manager                          │
│  • System Information (OS, shell, environment)              │
│  • Working Directory Context                                │
│  • Command History                                          │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                     LLM Provider                            │
│  ┌──────────────┐              ┌──────────────┐            │
│  │  LM Studio   │              │    Gemini    │            │
│  │   (Local)    │              │   (Cloud)    │            │
│  └──────────────┘              └──────────────┘            │
│  • JSON Response Parsing & Repair                          │
│  • Confidence Scoring                                       │
│  • Execution Policy Generation                              │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                    Safety Checker                           │
│  • Risk Classification (Low/Medium/High/Blocked)            │
│  • Pattern Matching (regex-based)                          │
│  • Install Command Detection                                │
│  • Confidence Threshold Validation                          │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                   Command Executor                          │
│  • Shell Command Execution                                  │
│  • Output Capture & Streaming                               │
│  • Error Handling                                           │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                    Built-in Tools                           │
│  • File Search (find, grep)                                 │
│  • Git Operations                                           │
│  • Web Search                                               │
│  • System Diagnostics                                       │
│  • File Index (workspace file tracking)                     │
└─────────────────────────────────────────────────────────────┘
```

### Core Components

#### 1. **Configuration (`internal/config/`)**
- Manages application settings with priority: env vars > config file > defaults
- Supports `~/.config/nsh/config.yml` for persistent configuration
- Handles provider-specific settings (LM Studio, Gemini)

#### 2. **Context Manager (`internal/context/`)**
- Gathers system information (OS, shell, environment variables)
- Maintains working directory context
- Provides relevant context to LLM for better command generation

#### 3. **LLM Provider (`internal/llm/`)**
- **Abstraction Layer**: Common interface for multiple LLM providers
- **LM Studio Provider**: Local, private inference via OpenAI-compatible API
- **Gemini Provider**: Cloud-based inference via Google's Gemini API
- **JSON Cleaning**: Robust parsing that handles imperfect LLM outputs
  - Extracts first valid JSON object/array
  - Repairs common issues (trailing commas, control chars, code fences)

#### 4. **Safety Checker (`internal/safety/`)**
- **Risk Classification**: Analyzes commands using regex patterns
- **Execution Policy**: Combines LLM suggestions with hardcoded rules
- **Install Detection**: Automatically identifies package manager commands
- **Confidence Validation**: Escalates to confirmation if confidence is low

#### 5. **Command Executor (`internal/executor/`)**
- Executes shell commands with proper error handling
- Captures and streams output to the user
- Supports dry-run mode for testing

#### 6. **File Index (`internal/fileindex/`)**
- **Build**: Creates workspace file index for fast lookups
- **Query**: Searches indexed files by name, extension, or path
- **Refresh**: Rebuilds index when files change
- **Watcher**: Monitors filesystem for changes (optional)
- **Types**: Defines index data structures

#### 7. **History (`internal/history/`)**
- Maintains command history for REPL mode
- Supports history navigation and replay (`:history`, `:again`)

#### 8. **Built-in Tools (`internal/tools/`)**
- File operations (find, grep)
- Git integration (status, log, diff)
- Web search capabilities
- System diagnostics
- Process and port management

#### 9. **UI Layer (`internal/ui/`)**
- **Markdown Rendering**: Uses `glamour` for rich terminal output
- **Animations**: Loading spinners and progress indicators
- **Styles**: Consistent color scheme and formatting
- **REPL Interface**: Interactive command-line interface

### Data Flow

1. **User Input** → UI Layer receives natural language or quick command
2. **Context Gathering** → System context and history are collected
3. **LLM Processing** → Provider generates command with execution policy
4. **Safety Check** → Command is analyzed for risk level
5. **Execution Decision** → Auto-execute or request confirmation
6. **Command Execution** → Shell command runs with output capture
7. **Result Display** → Formatted output shown to user
8. **History Update** → Command saved for future reference

### Key Design Principles

- **Safety First**: Multiple layers of protection prevent dangerous operations
- **Local-First**: Default to LM Studio for privacy and offline capability
- **Extensibility**: Provider abstraction allows easy addition of new LLMs
- **User Control**: Transparent execution with clear explanations
- **Robustness**: Handles imperfect LLM outputs gracefully

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