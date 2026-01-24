package safety

import (
	"regexp"
	"strings"

	"github.com/riddhishganeshmahajan/nsh/internal/config"
)

type CommandInfo interface {
	GetCommand() string
	GetRiskHints() []string
}

type RiskLevel string

const (
	RiskLow     RiskLevel = "Low"
	RiskMedium  RiskLevel = "Medium"
	RiskHigh    RiskLevel = "High"
	RiskBlocked RiskLevel = "Blocked"
)

type SafetyResult struct {
	Risk            RiskLevel
	Reasons         []string
	RequiresConfirm bool
	AllowExecute    bool
}

func Check(gen CommandInfo, cfg config.Config) SafetyResult {
	cmd := gen.GetCommand()
	if cmd == "" {
		return SafetyResult{
			Risk:            RiskBlocked,
			Reasons:         []string{"No command generated"},
			RequiresConfirm: false,
			AllowExecute:    false,
		}
	}

	result := SafetyResult{
		Risk:         RiskLow,
		AllowExecute: true,
	}

	for _, pattern := range cfg.Safety.BlockPatterns {
		if matched, _ := regexp.MatchString(pattern, cmd); matched {
			result.Risk = RiskBlocked
			result.Reasons = append(result.Reasons, "Matches blocked pattern: "+pattern)
			result.AllowExecute = false
			result.RequiresConfirm = false
			return result
		}
	}

	for _, pattern := range cfg.Safety.HighPatterns {
		if matched, _ := regexp.MatchString(pattern, cmd); matched {
			if result.Risk != RiskBlocked {
				result.Risk = RiskHigh
			}
			result.Reasons = append(result.Reasons, "Matches high-risk pattern: "+pattern)
		}
	}

	if result.Risk != RiskHigh && result.Risk != RiskBlocked {
		for _, pattern := range cfg.Safety.MediumPatterns {
			if matched, _ := regexp.MatchString(pattern, cmd); matched {
				result.Risk = RiskMedium
				result.Reasons = append(result.Reasons, "Matches medium-risk pattern: "+pattern)
			}
		}
	}

	if strings.Contains(cmd, "|") && (strings.Contains(cmd, "sh") || strings.Contains(cmd, "bash")) {
		if result.Risk == RiskLow || result.Risk == RiskMedium {
			result.Risk = RiskHigh
			result.Reasons = append(result.Reasons, "Contains pipe to shell execution")
		}
	}

	if strings.Contains(cmd, "$(") || strings.Contains(cmd, "`") {
		if result.Risk == RiskLow {
			result.Risk = RiskMedium
			result.Reasons = append(result.Reasons, "Contains command substitution")
		}
	}

	for _, hint := range gen.GetRiskHints() {
		if containsRiskyKeyword(hint) {
			if result.Risk == RiskLow {
				result.Risk = RiskMedium
			}
			result.Reasons = append(result.Reasons, "Model warning: "+hint)
		}
	}

	// Check for software installation commands - always require confirmation
	if isInstall, reason := isInstallCommand(cmd); isInstall {
		if result.Risk == RiskLow {
			result.Risk = RiskMedium
		}
		result.Reasons = append(result.Reasons, "Software installation: "+reason)
		result.RequiresConfirm = true
	}

	switch result.Risk {
	case RiskBlocked:
		result.RequiresConfirm = false
		result.AllowExecute = false
	case RiskHigh:
		result.RequiresConfirm = cfg.UI.ConfirmHigh
		result.AllowExecute = true
	case RiskMedium:
		result.RequiresConfirm = cfg.UI.ConfirmMedium
		result.AllowExecute = true
	case RiskLow:
		result.RequiresConfirm = cfg.UI.AlwaysConfirm
		result.AllowExecute = true
	}

	return result
}

func containsRiskyKeyword(s string) bool {
	keywords := []string{
		"delete", "remove", "destroy", "overwrite", "dangerous",
		"irreversible", "permanent", "loss", "wipe", "force",
	}
	lower := strings.ToLower(s)
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// Install command patterns - these always require confirmation
var installPatterns = []struct {
	pattern *regexp.Regexp
	reason  string
}{
	{regexp.MustCompile(`(?i)(^|[;&|]\s*)(sudo\s+)?brew\s+install\b`), "Homebrew install"},
	{regexp.MustCompile(`(?i)(^|[;&|]\s*)(sudo\s+)?apt(-get)?\s+install\b`), "APT install"},
	{regexp.MustCompile(`(?i)(^|[;&|]\s*)(sudo\s+)?(yum|dnf)\s+install\b`), "YUM/DNF install"},
	{regexp.MustCompile(`(?i)(^|[;&|]\s*)(sudo\s+)?pacman\s+-S\b`), "Pacman install"},
	{regexp.MustCompile(`(?i)(^|[;&|]\s*)(sudo\s+)?apk\s+add\b`), "APK add"},
	{regexp.MustCompile(`(?i)(^|[;&|]\s*)(sudo\s+)?snap\s+install\b`), "Snap install"},
	{regexp.MustCompile(`(?i)(^|[;&|]\s*)(sudo\s+)?flatpak\s+install\b`), "Flatpak install"},
	{regexp.MustCompile(`(?i)(^|[;&|]\s*)pip3?\s+install\b`), "pip install"},
	{regexp.MustCompile(`(?i)(^|[;&|]\s*)npm\s+(install|i)\s`), "npm install"},
	{regexp.MustCompile(`(?i)(^|[;&|]\s*)(yarn|pnpm)\s+(add|install)\b`), "yarn/pnpm install"},
	{regexp.MustCompile(`(?i)(^|[;&|]\s*)go\s+install\b`), "go install"},
	{regexp.MustCompile(`(?i)(^|[;&|]\s*)cargo\s+install\b`), "cargo install"},
	{regexp.MustCompile(`(?i)(^|[;&|]\s*)gem\s+install\b`), "gem install"},
	{regexp.MustCompile(`(?i)(^|[;&|]\s*)composer\s+(require|install)\b`), "composer install"},
	{regexp.MustCompile(`(?i)(^|[;&|]\s*)conda\s+install\b`), "conda install"},
	{regexp.MustCompile(`(?i)(^|[;&|]\s*)port\s+install\b`), "MacPorts install"},
	{regexp.MustCompile(`(?i)(^|[;&|]\s*)nix-env\s+-i\b`), "Nix install"},
	{regexp.MustCompile(`(?i)(^|[;&|]\s*)winget\s+install\b`), "winget install"},
	{regexp.MustCompile(`(?i)(^|[;&|]\s*)choco\s+install\b`), "Chocolatey install"},
	{regexp.MustCompile(`(?i)(^|[;&|]\s*)scoop\s+install\b`), "Scoop install"},
}

func isInstallCommand(cmd string) (bool, string) {
	for _, p := range installPatterns {
		if p.pattern.MatchString(cmd) {
			return true, p.reason
		}
	}
	return false, ""
}

func (r RiskLevel) Color() string {
	switch r {
	case RiskLow:
		return "green"
	case RiskMedium:
		return "yellow"
	case RiskHigh:
		return "red"
	case RiskBlocked:
		return "magenta"
	default:
		return "white"
	}
}
