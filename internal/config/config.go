package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Provider string         `yaml:"provider"` // "gemini" or "lmstudio"
	Gemini   GeminiConfig   `yaml:"gemini"`
	LMStudio LMStudioConfig `yaml:"lmstudio"`
	UI       UIConfig       `yaml:"ui"`
	Safety   SafetyConfig   `yaml:"safety"`
	Exec     ExecConfig     `yaml:"exec"`
	History  HistoryConfig  `yaml:"history"`
}

type LMStudioConfig struct {
	BaseURL        string `yaml:"base_url"`
	Model          string `yaml:"model"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

type GeminiConfig struct {
	APIKeyEnv      string `yaml:"api_key_env"`
	Model          string `yaml:"model"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

type UIConfig struct {
	AlwaysConfirm         bool    `yaml:"always_confirm"`
	ConfirmMedium         bool    `yaml:"confirm_medium"`
	ConfirmHigh           bool    `yaml:"confirm_high"`
	Color                 bool    `yaml:"color"`
	LearnMode             bool    `yaml:"learn_mode"`
	MinAutoExecConfidence float64 `yaml:"min_auto_exec_confidence"`
}

type SafetyConfig struct {
	BlockPatterns  []string `yaml:"block_patterns"`
	HighPatterns   []string `yaml:"high_patterns"`
	MediumPatterns []string `yaml:"medium_patterns"`
}

type ExecConfig struct {
	DryRun        bool `yaml:"dry_run"`
	ConfirmMode   bool `yaml:"confirm_mode"`
	UseLoginShell bool `yaml:"use_login_shell"`
}

type HistoryConfig struct {
	MaxEntries int `yaml:"max_entries"`
}

func DefaultConfig() Config {
	return Config{
		Provider: "lmstudio", // Default to local LM Studio
		Gemini: GeminiConfig{
			APIKeyEnv:      "GEMINI_API_KEY",
			Model:          "gemini-2.5-flash",
			TimeoutSeconds: 30,
		},
		LMStudio: LMStudioConfig{
			BaseURL:        "http://localhost:1234/v1", // Standard LM Studio port
			Model:          "local-model",              // Will use whatever model is loaded
			TimeoutSeconds: 120,
		},
		UI: UIConfig{
			AlwaysConfirm:         false,
			ConfirmMedium:         true,
			ConfirmHigh:           true,
			Color:                 true,
			LearnMode:             false,
			MinAutoExecConfidence: 0.8,
		},
		Safety: SafetyConfig{
			BlockPatterns: []string{
				`rm\s+-rf\s+/\s*$`,
				`rm\s+-rf\s+~`,
				`rm\s+-rf\s+\$HOME`,
				`:\(\)\{\s*:\|:&\s*\};:`,
				`mkfs\.`,
				`dd\s+if=.*/dev/`,
				`curl.*\|\s*(sh|bash|zsh)`,
				`wget.*\|\s*(sh|bash|zsh)`,
			},
			HighPatterns: []string{
				`\brm\b.*-rf\b`,
				`\bgit\b.*reset\s+--hard`,
				`\bgit\b.*clean\s+-fdx`,
				`\bgit\b.*push.*--force`,
				`\bchmod\b.*-R`,
				`\bchown\b.*-R`,
				`\bsudo\b`,
			},
			MediumPatterns: []string{
				`\bcurl\b`,
				`\bwget\b`,
				`\bssh\b`,
				`\bscp\b`,
				`\brsync\b`,
				`\brm\b`,
				`\bgit\b.*push`,
				`\bgit\b.*commit`,
				`\bgit\b.*rebase`,
			},
		},
		Exec: ExecConfig{
			DryRun:        false,
			ConfirmMode:   false,
			UseLoginShell: false,
		},
		History: HistoryConfig{
			MaxEntries: 50,
		},
	}
}

func configPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "nsh", "config.yml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "nsh", "config.yml")
}

func Load() (Config, error) {
	cfg := DefaultConfig()

	path := configPath()
	data, err := os.ReadFile(path)
	if err == nil {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return cfg, err
		}
	} else if !os.IsNotExist(err) {
		return cfg, err
	}

	applyEnvOverrides(&cfg)

	return cfg, nil
}

// applyEnvOverrides applies environment variable overrides
// Priority: env vars > config file > defaults
//
// Supported environment variables:
//   - NSH_PROVIDER: "lmstudio" or "gemini"
//   - NSH_LMSTUDIO_URL: LM Studio API URL (e.g., "http://localhost:1234/v1")
//   - NSH_LMSTUDIO_MODEL: Model name (e.g., "deepseek/deepseek-r1-0528-qwen3-8b")
//   - NSH_LMSTUDIO_TIMEOUT: Timeout in seconds
//   - NSH_GEMINI_MODEL: Gemini model name
//   - NSH_GEMINI_TIMEOUT: Gemini timeout in seconds
//   - NSH_DRY_RUN: true|false
//   - NSH_CONFIRM_MODE: true|false
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("NSH_PROVIDER"); v != "" {
		cfg.Provider = v
	}

	if v := os.Getenv("NSH_LMSTUDIO_URL"); v != "" {
		cfg.LMStudio.BaseURL = v
	}
	if v := os.Getenv("NSH_LMSTUDIO_MODEL"); v != "" {
		cfg.LMStudio.Model = v
	}
	if v := os.Getenv("NSH_LMSTUDIO_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.LMStudio.TimeoutSeconds = n
		}
	}

	if v := os.Getenv("NSH_GEMINI_MODEL"); v != "" {
		cfg.Gemini.Model = v
	}
	if v := os.Getenv("NSH_GEMINI_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Gemini.TimeoutSeconds = n
		}
	}

	if v := os.Getenv("NSH_DRY_RUN"); v != "" {
		if b, ok := parseBool(v); ok {
			cfg.Exec.DryRun = b
		}
	}
	if v := os.Getenv("NSH_CONFIRM_MODE"); v != "" {
		if b, ok := parseBool(v); ok {
			cfg.Exec.ConfirmMode = b
		}
	}
}

func parseBool(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true, true
	case "0", "false", "no", "n", "off":
		return false, true
	default:
		return false, false
	}
}

func Save(cfg Config) error {
	path := configPath()
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func GetAPIKey(cfg Config) string {
	return os.Getenv(cfg.Gemini.APIKeyEnv)
}