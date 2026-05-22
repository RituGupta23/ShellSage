// Package config handles loading, validation, and resolution of ShellSage configuration
// from config files, environment variables, and CLI flags.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const (
	// DefaultProvider is the default AI provider used when none is configured.
	DefaultProvider = "gemini"
	// DefaultModel is the default Gemini model used when none is configured.
	DefaultModel = "gemini-2.5-flash"
	// DefaultMode is the default execution mode (dry = display only).
	DefaultMode = "dry"

	// ConfigDirName is the configuration directory under the user's home.
	ConfigDirName = ".shellsage"
	// ConfigFileName is the TOML config file name (without extension).
	ConfigFileName = "config"
	// ConfigFileType is the config file format viper should parse.
	ConfigFileType = "toml"
	// PromptOverrideFile is the optional user-supplied system prompt file.
	PromptOverrideFile = "prompts/system.txt"
)

// Config is the resolved, validated configuration for a ShellSage run.
type Config struct {
	Provider ProviderConfig
	Behavior BehaviorConfig
	Prompt   PromptConfig
}

// ProviderConfig holds AI provider settings.
type ProviderConfig struct {
	Name    string // claude | openai | ollama
	Model   string
	APIKey  string
	BaseURL string // used for ollama
}

// BehaviorConfig controls runtime behavior.
type BehaviorConfig struct {
	DefaultMode  string // dry | run
	ConfirmRisky bool
	OSOverride   string
	NoColor      bool
}

// PromptConfig holds system prompt settings.
type PromptConfig struct {
	OverridePath string
}

// Load initialises Viper with the config file, binds environment variables,
// and returns a fully resolved Config. It does NOT require a config file to exist.
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults so the binary works without any config file.
	setDefaults(v)

	// Locate and load the config file if it exists.
	configDir, err := configDirPath()
	if err != nil {
		return nil, fmt.Errorf("config: resolving home directory: %w", err)
	}
	v.SetConfigName(ConfigFileName)
	v.SetConfigType(ConfigFileType)
	v.AddConfigPath(configDir)

	if err := v.ReadInConfig(); err != nil {
		// Missing config file is acceptable — defaults + env vars are sufficient.
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("config: reading config file: %w", err)
		}
	}

	// Bind environment variables. Precedence: env > config file > defaults.
	bindEnvVars(v)

	cfg := &Config{}
	if err := unmarshal(v, cfg, configDir); err != nil {
		return nil, fmt.Errorf("config: unmarshalling: %w", err)
	}

	return cfg, nil
}

// ConfigDirPath returns the path to ~/.shellsage.
func ConfigDirPath() (string, error) {
	return configDirPath()
}

func configDirPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ConfigDirName), nil
}

// ConfigFilePath returns the full path to the config TOML file.
func ConfigFilePath() (string, error) {
	dir, err := configDirPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ConfigFileName+"."+ConfigFileType), nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("provider.name", DefaultProvider)
	v.SetDefault("provider.model", DefaultModel)
	v.SetDefault("provider.api_key", "")
	v.SetDefault("provider.base_url", "")

	v.SetDefault("behavior.default_mode", DefaultMode)
	v.SetDefault("behavior.confirm_risky", true)
	v.SetDefault("behavior.os_override", "")
	v.SetDefault("behavior.no_color", false)

	v.SetDefault("prompt.override_path", "")
}

func bindEnvVars(v *viper.Viper) {
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Provider-specific API key env vars take highest priority.
	// We resolve these manually in unmarshal since viper can't map two env vars to one key.
	_ = v.BindEnv("provider.name", "SHELLSAGE_PROVIDER")
}

// unmarshal reads viper values into cfg, applying the documented env var priority:
// ANTHROPIC_API_KEY / OPENAI_API_KEY > SHELLSAGE_API_KEY > config file.
func unmarshal(v *viper.Viper, cfg *Config, configDir string) error {
	cfg.Provider = ProviderConfig{
		Name:    v.GetString("provider.name"),
		Model:   v.GetString("provider.model"),
		APIKey:  v.GetString("provider.api_key"),
		BaseURL: v.GetString("provider.base_url"),
	}
	cfg.Behavior = BehaviorConfig{
		DefaultMode:  v.GetString("behavior.default_mode"),
		ConfirmRisky: v.GetBool("behavior.confirm_risky"),
		OSOverride:   v.GetString("behavior.os_override"),
		NoColor:      v.GetBool("behavior.no_color"),
	}
	cfg.Prompt = PromptConfig{
		OverridePath: v.GetString("prompt.override_path"),
	}

	// Resolve API key using priority chain.
	cfg.Provider.APIKey = ResolveAPIKey(cfg.Provider.Name, cfg.Provider.APIKey)

	// Resolve user system prompt override: config path > ~/.shellsage/prompts/system.txt.
	cfg.Prompt.OverridePath = resolvePromptOverride(cfg.Prompt.OverridePath, configDir)

	return nil
}

// ResolveAPIKey implements the documented priority chain for API keys.
// It is exported so the CLI layer can re-resolve the key after a --provider flag override.
func ResolveAPIKey(providerName, configFileKey string) string {
	// Priority 1: provider-specific env var.
	switch strings.ToLower(providerName) {
	case "claude":
		if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
			return v
		}
	case "openai":
		if v := os.Getenv("OPENAI_API_KEY"); v != "" {
			return v
		}
	case "gemini":
		if v := os.Getenv("GEMINI_API_KEY"); v != "" {
			return v
		}
	}

	// Priority 2: generic fallback env var.
	if v := os.Getenv("SHELLSAGE_API_KEY"); v != "" {
		return v
	}

	// Priority 3 / 4: whatever came from the config file or defaults.
	return configFileKey
}

// resolvePromptOverride returns the first existing prompt path.
func resolvePromptOverride(configuredPath, configDir string) string {
	// Explicit path from config takes priority.
	if configuredPath != "" {
		if _, err := os.Stat(configuredPath); err == nil {
			return configuredPath
		}
	}

	// Fall back to the conventional location under ~/.shellsage/.
	conventional := filepath.Join(configDir, PromptOverrideFile)
	if _, err := os.Stat(conventional); err == nil {
		return conventional
	}

	return ""
}

// RedactedAPIKey returns a redacted version of the API key safe for display.
func RedactedAPIKey(key string) string {
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}

// DefaultConfigTOML returns the content to write for a fresh config.toml.
func DefaultConfigTOML(providerName, model, apiKey, baseURL string) string {
	return fmt.Sprintf(`[provider]
  name     = %q
  model    = %q
  api_key  = %q
  base_url = %q

[behavior]
  default_mode  = "dry"
  confirm_risky = true
  os_override   = ""
  no_color      = false

[prompt]
  override_path = ""
`, providerName, model, apiKey, baseURL)
}
