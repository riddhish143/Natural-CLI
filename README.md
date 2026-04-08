# nsh — Natural Shell
<img width="1352" height="508" alt="image" src="https://github.com/user-attachments/assets/9ff8d7fb-d6c1-4a89-b077-313972c1ad32" />

Natural language shell that translates plain English into executable commands with built-in safety controls.

**Local-first:** Uses LM Studio (local, private) by default, with Google Gemini as cloud alternative.

---

## Quick Start

```bash
# One-shot
nsh "find large files in this folder"

# Interactive REPL
nsh
```

---

## Architecture

```
User Input → UI Layer → Context Manager → LLM Provider → Safety Checker → Executor
                                              ↓
                                        Built-in Tools
```

**Core Components:**
- **Config** (`internal/config/`): Settings management (env vars > config file > defaults)
- **Context** (`internal/context/`): System info, working directory, command history
- **LLM** (`internal/llm/`): Provider abstraction (LM Studio, Gemini), JSON parsing
- **Safety** (`internal/safety/`): Risk classification, install detection, confidence validation
- **Executor** (`internal/executor/`): Shell command execution with error handling
- **File Index** (`internal/fileindex/`): Fast workspace file lookups
- **Tools** (`internal/tools/`): File ops, git, web search, diagnostics
- **UI** (`internal/ui/`): Markdown rendering, animations, REPL interface

**Design Principles:** Safety First • Local-First • Extensible • User Control • Robust

---

## Installation

```bash
git clone https://github.com/riddhishganeshmahajan/nsh.git
cd nsh
make install
```

Or:
```bash
go install github.com/riddhishganeshmahajan/nsh/cmd/nsh@latest
```

---

## Setup

### LM Studio (Default)

1. Install from https://lmstudio.ai
2. Load a model and start server
3. Run: `nsh --setup`

```bash
export NSH_PROVIDER=lmstudio
export NSH_LMSTUDIO_URL="http://localhost:1234/v1"
```

### Gemini (Alternative)

```bash
export GEMINI_API_KEY="your-key"
export NSH_PROVIDER=gemini
```

---

## Usage

### Commands

```bash
nsh "show disk usage"
nsh "find all python files"
nsh "what's using port 8080"
```

### REPL Mode

```bash
nsh
> :help          # Show help
> :history       # Command history
> :git           # Git status
> :find *.go     # Find files
> :dry           # Toggle dry-run
> exit           # Exit
```

### Flags

- `--dry-run`, `-n`: Preview command without executing (dry-run mode)
- `--confirm`: Always ask for confirmation before executing
- `--learn`: Show explanations/alternatives
- `--setup`: Interactive setup wizard
- `--force`: Override blocks (dangerous)
- `--version`: Print version

**Examples:**
```bash
nsh --dry-run "delete all .log files"
nsh --confirm "install docker"
nsh --learn "find process on port 3000"
```

---

## Safety

**Auto-executes:** Read-only commands (`ls`, `cat`, `find`, `ps`)

**Requires confirmation:**
- File modifications (`rm`, `mv`, `cp`)
- `sudo` commands
- Destructive git ops (`reset --hard`, `push --force`)
- Package installs (auto-detected: `brew`, `apt`, `pip`, `npm`, etc.)

**Risk Levels:**
- **Low**: Auto-execute if safe
- **Medium**: Confirm by default
- **High**: Explicit confirm
- **Blocked**: Never execute (unless `--force`)

**Priority Order:**
1. `--dry-run` (never executes)
2. `--confirm` (always asks)
3. Safety-based confirmation (risk dependent)
4. Auto-execution (safe commands only)

---

## Configuration

**Priority:** env vars > `~/.config/nsh/config.yml` > defaults

**Key Variables:**
```bash
NSH_PROVIDER=lmstudio              # or "gemini"
NSH_DRY_RUN=true                   # Enable dry-run globally
NSH_CONFIRM=true                   # Enable confirm globally
NSH_LMSTUDIO_URL=http://localhost:1234/v1
NSH_LMSTUDIO_MODEL=local-model
NSH_GEMINI_MODEL=gemini-2.5-flash
GEMINI_API_KEY=your-key
```

**Config File:** `~/.config/nsh/config.yml`
```yaml
provider: lmstudio
lmstudio:
  base_url: "http://localhost:1234/v1"
  model: "local-model"
  timeout_seconds: 120
exec:
  dry_run: false
  confirm: false
ui:
  min_auto_exec_confidence: 0.8
```

---

## Features

- Natural language → shell commands
- REPL with quick commands (`:help`, `:history`, `:git`)
- Built-in tools (file search, git, web search)
- Risk classification (Low/Medium/High/Blocked)
- Install command detection
- LLM-driven execution policy
- Dry-run and confirm modes
- Markdown terminal output
- Robust JSON parsing
- Workspace file index

---

## Troubleshooting

**LM Studio:** Ensure running with model loaded at `http://localhost:1234/v1`

**Gemini:** Set `GEMINI_API_KEY` and `NSH_PROVIDER=gemini`

**JSON Errors:** Robust parser handles most cases; try different model if issues persist

---

## Security

- Blocked patterns never execute (unless `--force`)
- Installs always require confirmation
- Low confidence escalates to confirmation
- **Always review commands before confirming**

---