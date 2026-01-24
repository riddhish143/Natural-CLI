package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Provider string        `yaml:"provider"` // "gemini" or "lmstudio"
	Gemini   GeminiConfig  `yaml:"gemini"`
	LMStudio LMStudioConfig `yaml:"lmstudio"`
	UI       UIConfig      `yaml:"ui"`
	Safety   SafetyConfig  `yaml:"safety"`
	Exec     ExecConfig    `yaml:"exec"`
	History  HistoryConfig `yaml:"history"`
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
	AlwaysConfirm        bool    `yaml:"always_confirm"`
	ConfirmMedium        bool    `yaml:"confirm_medium"`
	ConfirmHigh          bool    `yaml:"confirm_high"`
	Color                bool    `yaml:"color"`
	LearnMode            bool    `yaml:"learn_mode"`
	MinAutoExecConfidence float64 `yaml:"min_auto_exec_confidence"`
}

type SafetyConfig struct {
	BlockPatterns  []string `yaml:"block_patterns"`
	HighPatterns   []string `yaml:"high_patterns"`
	MediumPatterns []string `yaml:"medium_patterns"`
}

type ExecConfig struct {
	DryRun        bool `yaml:"dry_run"`
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
			BaseURL:        "http://localhost:1234/v1",
			Model:          "openai/gpt-oss-20b",
			TimeoutSeconds: 120,
		},
		UI: UIConfig{
			AlwaysConfirm:        false,
			ConfirmMedium:        true,
			ConfirmHigh:          true,
			Color:                true,
			LearnMode:            false,
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
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
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
