# ShellSage (`sg`)

ShellSage translates plain English into shell commands using AI. Describe what you want to do — ShellSage generates the correct command for your OS, classifies how dangerous it is, and optionally runs it after your confirmation.

```
sg "find all log files modified in the last 3 days"

  macOS / Linux   →  find . -name "*.log" -mtime -3
  Windows         →  Get-ChildItem -Recurse -Filter *.log | Where-Object { $_.LastWriteTime -gt (Get-Date).AddDays(-3) }

  Run? [y/N]
```

---

## Table of Contents

- [Supported AI Providers](#supported-ai-providers)
- [Installation](#installation)
  - [Option 1 — Pre-built Binary (no Go required)](#option-1--pre-built-binary-no-go-required)
  - [Option 2 — go install (requires Go)](#option-2--go-install-requires-go)
  - [Option 3 — Build from Source](#option-3--build-from-source)
- [First-time Setup](#first-time-setup)
- [Usage](#usage)
- [Flags](#flags)
- [Configuration](#configuration)
- [Risk Classification](#risk-classification)
- [Use as a Go Library](#use-as-a-go-library)

---

## Supported AI Providers

| Provider | Default Model | Free Tier | API Key Env Var |
|---|---|---|---|
| **Gemini** (default) | `gemini-2.5-flash` | Yes | `GEMINI_API_KEY` |
| **Claude** | `claude-sonnet-4-20250514` | No | `ANTHROPIC_API_KEY` |
| **OpenAI** | `gpt-4o` | No | `OPENAI_API_KEY` |
| **Ollama** | `llama3` | Local, free | none |

---

## Installation

### Option 1 — Pre-built Binary (no Go required)

Download a pre-compiled binary from the [Releases page](https://github.com/RituGupta23/ShellSage/releases). No Go installation needed.

---

#### macOS

**Step 1 — Find your architecture**

```bash
uname -m
# arm64  → Apple Silicon (M1/M2/M3/M4) → download sg_darwin_arm64
# x86_64 → Intel Mac                    → download sg_darwin_amd64
```

**Step 2 — Download** the correct file from the [latest release](https://github.com/RituGupta23/ShellSage/releases/latest).

**Step 3 — Install**

```bash
cd ~/Downloads

# Make it executable (replace arm64 with amd64 if Intel)
chmod +x sg_darwin_arm64

# Move to /usr/local/bin so it's available system-wide
sudo mv sg_darwin_arm64 /usr/local/bin/sg
```

**Step 4 — Remove macOS security quarantine**

macOS blocks unsigned binaries by default. Run this once:

```bash
xattr -d com.apple.quarantine /usr/local/bin/sg
```

If you see a popup instead, go to **System Settings → Privacy & Security** and click **Allow Anyway**.

**Step 5 — Verify**

```bash
sg version
```

---

#### Linux

**Step 1 — Download** `sg_linux_amd64` from the [latest release](https://github.com/RituGupta23/ShellSage/releases/latest).

**Step 2 — Install**

```bash
cd ~/Downloads
chmod +x sg_linux_amd64
sudo mv sg_linux_amd64 /usr/local/bin/sg
```

**Step 3 — Verify**

```bash
sg version
```

If `/usr/local/bin` is not in your PATH:

```bash
echo 'export PATH="$PATH:/usr/local/bin"' >> ~/.bashrc
source ~/.bashrc
```

---

#### Windows

**Step 1 — Download** `sg_windows_amd64.exe` from the [latest release](https://github.com/RituGupta23/ShellSage/releases/latest).

**Step 2 — Install**

Option A — copy to System32 (available everywhere, requires admin):
```powershell
# In PowerShell as Administrator
Copy-Item "$env:USERPROFILE\Downloads\sg_windows_amd64.exe" "C:\Windows\System32\sg.exe"
```

Option B — create a personal bin folder and add it to PATH:
```powershell
# Create folder
New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\bin"

# Copy and rename the binary
Copy-Item "$env:USERPROFILE\Downloads\sg_windows_amd64.exe" "$env:USERPROFILE\bin\sg.exe"

# Add to PATH permanently (run once)
[Environment]::SetEnvironmentVariable(
  "PATH",
  $env:PATH + ";$env:USERPROFILE\bin",
  [EnvironmentVariableTarget]::User
)

# Reload PATH in current session
$env:PATH += ";$env:USERPROFILE\bin"
```

**Step 3 — Verify** (open a new PowerShell window)

```powershell
sg version
```

---

### Option 2 — `go install` (requires Go)

Requires [Go 1.21+](https://go.dev/dl/).

```bash
go install github.com/RituGupta23/ShellSage/cmd@latest
```

The binary will be placed in `~/go/bin/`. Make sure that directory is in your PATH.

**macOS / Linux:**
```bash
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc   # or ~/.bashrc
source ~/.zshrc
```

**Windows (PowerShell):**
```powershell
$gopath = go env GOPATH
[Environment]::SetEnvironmentVariable("PATH", $env:PATH + ";$gopath\bin", [EnvironmentVariableTarget]::User)
$env:PATH += ";$gopath\bin"
```

**Note:** The installed binary will be named `cmd`. Rename it to `sg`:

```bash
# macOS / Linux
mv "$(go env GOPATH)/bin/cmd" "$(go env GOPATH)/bin/sg"

# Windows PowerShell
Rename-Item "$env:GOPATH\bin\cmd.exe" "sg.exe"
```

**Verify:**
```bash
sg version
```

---

### Option 3 — Build from Source

Requires [Go 1.21+](https://go.dev/dl/) and `make`.

**macOS / Linux:**

```bash
git clone https://github.com/RituGupta23/ShellSage.git
cd ShellSage

# Build and install to ~/go/bin/sg
make install

# OR just build locally to ./dist/sg
make build
./dist/sg version
```

**Windows (PowerShell):**

```powershell
git clone https://github.com/RituGupta23/ShellSage.git
cd ShellSage

# Build manually (no make on Windows by default)
go build -trimpath -o dist\sg.exe .\cmd

.\dist\sg.exe version
```

To install the Windows build globally:
```powershell
Copy-Item "dist\sg.exe" "C:\Windows\System32\sg.exe"
```

**Available make targets:**

| Command | Description |
|---|---|
| `make build` | Compile binary to `./dist/sg` |
| `make install` | Build and install to `~/go/bin/sg` |
| `make test` | Run all tests with race detector |
| `make lint` | Run golangci-lint |
| `make clean` | Remove build artifacts |

---

## First-time Setup

After installing, run the interactive setup wizard once:

```bash
sg config init
```

Example session:

```
ShellSage Configuration Wizard
────────────────────────────────────────
AI provider [claude/openai/ollama/gemini] (default: claude): gemini
Model (default: gemini-2.5-flash):
API key (detected GEMINI_API_KEY in environment — press Enter to use it):
Default mode [dry/run] (default: dry):
Validating API key with gemini... OK

Config written to /Users/yourname/.shellsage/config.toml
You can now run: sg "your query here"
```

This writes `~/.shellsage/config.toml` (on Windows: `C:\Users\yourname\.shellsage\config.toml`).

### Skip the wizard — use environment variables

Set your API key as an environment variable and ShellSage works immediately with no config file.

**macOS / Linux:**
```bash
# Add to ~/.zshrc or ~/.bashrc
export GEMINI_API_KEY="your-key-here"

# Reload
source ~/.zshrc
```

**Windows (PowerShell):**
```powershell
# Set permanently for your user
[Environment]::SetEnvironmentVariable("GEMINI_API_KEY", "your-key-here", [EnvironmentVariableTarget]::User)

# Apply in current session
$env:GEMINI_API_KEY = "your-key-here"
```

### Getting API Keys

| Provider | Where to get it |
|---|---|
| Gemini | [aistudio.google.com/apikey](https://aistudio.google.com/apikey) — free tier available |
| Claude | [console.anthropic.com](https://console.anthropic.com) |
| OpenAI | [platform.openai.com/api-keys](https://platform.openai.com/api-keys) |
| Ollama | No key needed — install Ollama locally from [ollama.com](https://ollama.com) |

---

## Usage

```bash
sg "your plain English query"
```

### Examples

```bash
# Show the command, then ask if you want to run it
sg "find all files larger than 100MB"

# Show command for a different OS without running
sg --dry --os linux "open port 8080 in the firewall"

# Generate and execute immediately (still confirms for risky commands)
sg --run "list all running processes sorted by memory"

# Use a specific provider for one query
sg --provider claude "compress the src folder into a tar.gz"

# Use a specific model
sg --model gemini-2.5-flash "show all open network connections"

# Check your current configuration
sg config show

# Re-run the setup wizard
sg config init

# Show version info
sg version
```

---

## Flags

| Flag | Description |
|---|---|
| `--run` | Execute the generated command after confirmation |
| `--dry` | Display the command only — never execute, even for low-risk commands |
| `--os <name>` | Override detected OS: `macos`, `linux`, or `windows` |
| `--provider <name>` | Override configured provider: `claude`, `openai`, `gemini`, `ollama` |
| `--model <name>` | Override configured model (e.g. `gpt-4o-mini`, `claude-haiku-4-20250514`) |
| `--no-color` | Disable colored output (plain text) |

Flags always take priority over the config file.

---

## Configuration

The config file lives at `~/.shellsage/config.toml` (created by `sg config init`).

```toml
[provider]
  name     = "gemini"                  # claude | openai | ollama | gemini
  model    = "gemini-2.5-flash"
  api_key  = "your-api-key-here"
  base_url = ""                        # only needed for ollama

[behavior]
  default_mode  = "dry"               # dry = display only | run = execute after confirm
  confirm_risky = true                 # extra confirmation for medium/high risk commands
  os_override   = ""                   # force a specific OS: macos | linux | windows
  no_color      = false

[prompt]
  override_path = ""                   # path to a custom system prompt file
```

### Configuration Priority (highest to lowest)

1. CLI flags (`--provider`, `--model`, `--os`, etc.)
2. Environment variables (`GEMINI_API_KEY`, `SHELLSAGE_PROVIDER`, etc.)
3. Config file (`~/.shellsage/config.toml`)
4. Built-in defaults

### Environment Variables

| Variable | Description |
|---|---|
| `GEMINI_API_KEY` | API key for Google Gemini |
| `ANTHROPIC_API_KEY` | API key for Anthropic Claude |
| `OPENAI_API_KEY` | API key for OpenAI |
| `SHELLSAGE_API_KEY` | Generic fallback API key for any provider |
| `SHELLSAGE_PROVIDER` | Override the default provider |
| `NO_COLOR` | Set to any value to disable colored output |

### Custom System Prompt

You can replace the built-in AI instructions with your own prompt file:

```toml
[prompt]
  override_path = "/path/to/my-prompt.txt"
```

Or place a file at `~/.shellsage/prompts/system.txt` — ShellSage picks it up automatically.

### Ollama (local AI, no API key)

Install [Ollama](https://ollama.com), pull a model, then configure ShellSage:

```bash
ollama pull llama3

# Either run the wizard
sg config init
# → choose "ollama", set base URL to http://localhost:11434

# Or set directly in config.toml
```

```toml
[provider]
  name     = "ollama"
  model    = "llama3"
  base_url = "http://localhost:11434"
```

---

## Risk Classification

Every generated command is classified before you see it. ShellSage runs both a local pattern-based classifier and asks the AI — and always uses the **more dangerous** of the two ratings.

| Level | What it means | Examples | Confirmation |
|---|---|---|---|
| **Low** | Read-only or informational | `ls`, `grep`, `cat`, `find`, `ps`, `ping` | `[y/N]` prompt |
| **Medium** | Mutates state but recoverable | `mv`, `cp -r`, `apt install`, `curl POST`, `systemctl` | Warning banner + `[y/N]` |
| **High** | Irreversible or destructive | `rm -rf`, `dd`, `mkfs`, `shutdown`, `chmod 777` | Must type **`yes`** in full |

---

## Use as a Go Library

ShellSage exposes a public API in `pkg/shellsage` for use in other Go programs — no CLI, no terminal UI, no command execution.

```bash
go get github.com/RituGupta23/ShellSage
```

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/RituGupta23/ShellSage/pkg/shellsage"
)

func main() {
    // Load config from ~/.shellsage/config.toml + env vars
    cfg, err := shellsage.LoadConfig()
    if err != nil {
        log.Fatal(err)
    }

    // Translate a plain English query into shell commands
    resp, err := shellsage.RunQuery(context.Background(), cfg, "list all docker containers")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp.Primary.Command)   // docker ps -a
    fmt.Println(resp.RiskLevel)         // low
    fmt.Println(resp.RiskReason)        // Read-only listing command.

    // Access all OS variants
    for _, v := range resp.Variants {
        fmt.Printf("%s (%s): %s\n", v.OS, v.Shell, v.Command)
    }
}
```

---

## License

MIT
