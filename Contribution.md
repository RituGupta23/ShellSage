
---

## Contributing

Contributions are welcome — bug fixes, new AI providers, better risk rules, docs improvements, and more.

### Getting Started

**Prerequisites:** Go 1.21+, Git, `make`

```bash
# Fork the repo on GitHub, then clone your fork
git clone https://github.com/YOUR_USERNAME/ShellSage.git
cd ShellSage

# Install dependencies
go mod tidy

# Build and verify everything works
make build
./dist/sg version
```

### Project Structure

```
cmd/
  main.go                   # Entry point — signal handling + CLI launch

internal/
  cli/
    root.go                 # Main `sg "query"` command and flag wiring
    config.go               # `sg config init` wizard and `sg config show`
    prompt.go               # System prompt resolution (bundled vs user override)
    version.go              # `sg version` command
  config/
    loader.go               # Viper-based config loader (~/.shellsage/config.toml)
  detector/
    os.go                   # Auto-detects OS and shell at runtime
  prompts/
    default.txt             # The system prompt sent to every AI provider
    embed.go                # Embeds default.txt into the binary at compile time
  provider/
    provider.go             # Provider interface + factory function + error types
    claude.go               # Anthropic Claude implementation
    openai.go               # OpenAI implementation
    gemini.go               # Google Gemini implementation
    ollama.go               # Local Ollama implementation
    parse.go                # JSON response parser shared by all providers
    message_helpers.go      # Shared request-building helpers
  risk/
    classifier.go           # Local pattern-based risk scorer (low / medium / high)
  ui/
    display.go              # Terminal rendering, confirmation prompts, execution
    styles.go               # lipgloss color styles

pkg/
  shellsage/
    shellsage.go            # Public Go library API (no UI, no execution)
```

### Development Workflow

```bash
# Run tests
make test

# Run tests with coverage report
make test-cover

# Lint (requires golangci-lint — brew install golangci-lint)
make lint

# Clean build artifacts
make clean
```

### Making Changes

1. **Create a branch** from `main`:
   ```bash
   git checkout -b feat/your-feature-name
   # or
   git checkout -b fix/what-you-are-fixing
   ```

2. **Make your changes** following the conventions below.

3. **Run tests** before committing:
   ```bash
   make test
   ```

4. **Commit** with a clear message:
   ```bash
   git commit -m "feat: add support for XYZ provider"
   git commit -m "fix: handle empty response from Gemini"
   git commit -m "docs: improve Windows installation steps"
   ```

5. **Push and open a Pull Request** against `main`.

### Adding a New AI Provider

All providers implement the `Provider` interface in `internal/provider/provider.go`:

```go
type Provider interface {
    Name() string
    GenerateCommand(ctx context.Context, req CommandRequest) (CommandResponse, error)
}
```

Steps to add a new provider (e.g. `mistral`):

1. Create `internal/provider/mistral.go` — implement the `Provider` interface.
2. Register it in the factory in `internal/provider/provider.go`:
   ```go
   case "mistral":
       return NewMistral(apiKey, model)
   ```
3. Add a default model in `defaultModelForProvider()` in `internal/cli/config.go`.
4. Add the API key env var in `apiKeyEnvVar()` in `internal/cli/config.go`.
5. Add it to the `ResolveAPIKey()` priority chain in `internal/config/loader.go`.
6. Update the wizard prompt in `runConfigInit()` in `internal/cli/config.go` to include the new provider name.

The AI must respond with the exact JSON schema defined in `internal/prompts/default.txt`. The shared `parseAIResponse()` function in `parse.go` handles the rest.

### Improving the Risk Classifier

The risk classifier lives in `internal/risk/classifier.go`. It works by:

1. Splitting chained commands (`;`, `&`)
2. Tokenizing each command
3. Scoring based on the base command, flags, and arguments
4. Returning `low` / `medium` / `high` based on the score

To add a new risky pattern, add a `case` in `classifySingle()`:

```go
case "your-command":
    score += 4
    reasons = append(reasons, "Reason this is dangerous.")
```

Score guide: 1–2 = low concern, 3–6 = medium, 7+ = high.

### Commit Message Convention

Use [Conventional Commits](https://www.conventionalcommits.org/):

| Prefix | When to use |
|---|---|
| `feat:` | New feature or new provider |
| `fix:` | Bug fix |
| `docs:` | Documentation only |
| `refactor:` | Code change that isn't a fix or feature |
| `test:` | Adding or fixing tests |
| `chore:` | Dependency updates, build changes |

### Pull Request Guidelines

- Keep PRs focused — one feature or fix per PR.
- Add or update tests for any changed logic.
- If you're adding a provider, include a short note in the PR about how to get an API key for it.
- Don't commit API keys, `.env` files, or any secrets.
- The `README.md` lives in the repo root — update it if your change affects user-facing behaviour.

### Reporting Bugs

Open an issue with:
- Your OS and shell
- The `sg version` output
- The query you ran
- The full error message or unexpected output
