# nsh - Natural Shell v2.0

A powerful AI-powered terminal assistant that understands natural language and can answer questions, search the web, analyze your system, and execute commands safely.

## Features

### 🧠 Intelligent Responses
- **Answer questions**: "what is kubernetes", "explain docker compose"
- **Execute commands**: "find all python files", "show disk usage"
- **Multi-step plans**: Automatically uses tools to gather info before responding

### 🔧 Built-in Tools
- **File operations**: Search, list, and read files
- **Content search**: Grep/ripgrep integration
- **System diagnostics**: CPU, memory, disk, processes, network
- **Process management**: Find processes, check ports
- **Package management**: Search and info for brew/apt/dnf/pacman
- **Web search**: Search the web for information
- **Git integration**: Status, log, diff

### 🛡️ Safety Features
- Risk classification (Low/Medium/High/Blocked)
- Confirmation prompts for dangerous commands
- Blocked patterns for destructive operations
- Sensitive file protection

### 📚 History & Context
- Command history with "do that again" support
- Context-aware (detects OS, shell, git, project type)
- Learning mode with command explanations

## Installation

### From Source (recommended)

```bash
# Clone the repository
git clone https://github.com/riddhishganeshmahajan/nsh.git
cd nsh

# Install with make
make install

# Or manually
go build -o nsh ./cmd/nsh
sudo cp nsh /usr/local/bin/
```

### Using Go Install

```bash
go install github.com/riddhishganeshmahajan/nsh/cmd/nsh@latest
```

### Setup

```bash
# Set API key for Gemini
export GEMINI_API_KEY='your-api-key'

# Or run setup wizard to configure LM Studio (local, free)
nsh --setup
```

## Usage

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

### REPL Mode
```bash
$ nsh
╭──────────────────────────────────────────╮
│  nsh - Natural Shell v2.0               │
│  Type naturally or use :help            │
╰──────────────────────────────────────────╯

nsh> what is docker
💡 Docker is a platform for developing, shipping, and running applications in containers...

nsh> show disk usage
Command: df -h
  Shows disk space usage in human-readable format.
  Risk: Low
```

### Quick Commands (REPL)
| Command | Description |
|---------|-------------|
| `:help` | Show help |
| `:history` | Command history |
| `:again` | Repeat last command |
| `:diag` | System diagnostics |
| `:git` | Git status |
| `:search <query>` | Web search |
| `:find <pattern>` | Find files |
| `:grep <pattern>` | Search contents |
| `:port <num>` | Check port |
| `:ps <name>` | Find processes |
| `:pkg <name>` | Search packages |
| `!<cmd>` | Raw shell command |
| `exit` | Exit nsh |

## Configuration

Config file: `~/.config/nsh/config.yml`

```yaml
gemini:
  api_key_env: GEMINI_API_KEY
  model: gemini-2.5-flash
  timeout_seconds: 30

ui:
  always_confirm: false
  confirm_medium: true
  confirm_high: true
  learn_mode: false

safety:
  block_patterns:
    - 'rm\s+-rf\s+/'
    - 'rm\s+-rf\s+~'
```

## Examples

```bash
# Questions
nsh "what is a kubernetes pod"
nsh "how do I create a python virtual environment"

# System analysis
nsh "is my system running low on memory"
nsh "what processes are using the most CPU"
nsh "is anything listening on port 5432"

# File operations
nsh "find all config files"
nsh "search for 'password' in all files"
nsh "what's in the readme file"

# Development
nsh "show git status"
nsh "list recent commits"
nsh "find all TODO comments"

# Web research
nsh "search for react hooks tutorial"
nsh "look up golang error handling best practices"
```

## Safety Levels

| Level | Behavior | Examples |
|-------|----------|----------|
| **Low** | Auto-execute | `ls`, `cat`, `git status` |
| **Medium** | Confirm | `rm file`, `curl`, `git push` |
| **High** | Explicit confirm | `rm -rf`, `git reset --hard` |
| **Blocked** | Never execute | `rm -rf /`, fork bombs |

