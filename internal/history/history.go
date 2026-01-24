package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Entry struct {
	Timestamp time.Time `json:"timestamp"`
	Input     string    `json:"input"`
	Command   string    `json:"command"`
	Mode      string    `json:"mode"`
	ExitCode  int       `json:"exit_code"`
	Output    string    `json:"output,omitempty"`
}

type History struct {
	entries    []Entry
	maxEntries int
	filePath   string
}

func New(maxEntries int) *History {
	home, _ := os.UserHomeDir()
	filePath := filepath.Join(home, ".config", "nsh", "history.json")

	h := &History{
		maxEntries: maxEntries,
		filePath:   filePath,
	}
	h.load()
	return h
}

func (h *History) Add(input, command, mode string, exitCode int, output string) {
	if len(output) > 500 {
		output = output[:500] + "..."
	}

	entry := Entry{
		Timestamp: time.Now(),
		Input:     input,
		Command:   command,
		Mode:      mode,
		ExitCode:  exitCode,
		Output:    output,
	}

	h.entries = append(h.entries, entry)

	if len(h.entries) > h.maxEntries {
		h.entries = h.entries[len(h.entries)-h.maxEntries:]
	}

	h.save()
}

func (h *History) Last() *Entry {
	if len(h.entries) == 0 {
		return nil
	}
	return &h.entries[len(h.entries)-1]
}

func (h *History) LastN(n int) []Entry {
	if n > len(h.entries) {
		n = len(h.entries)
	}
	return h.entries[len(h.entries)-n:]
}

func (h *History) GetCommand(index int) string {
	if index < 0 || index >= len(h.entries) {
		return ""
	}
	return h.entries[index].Command
}

func (h *History) Summary(n int) string {
	entries := h.LastN(n)
	if len(entries) == 0 {
		return "No command history"
	}

	var summary string
	for i, e := range entries {
		status := "✓"
		if e.ExitCode != 0 {
			status = "✗"
		}
		summary += "\n" + formatIndex(i) + ". " + status + " " + e.Input + " → " + e.Command
	}
	return summary
}

func (h *History) load() {
	data, err := os.ReadFile(h.filePath)
	if err != nil {
		return
	}
	json.Unmarshal(data, &h.entries)
}

func (h *History) save() {
	dir := filepath.Dir(h.filePath)
	os.MkdirAll(dir, 0755)

	data, err := json.MarshalIndent(h.entries, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(h.filePath, data, 0644)
}

func formatIndex(i int) string {
	if i < 10 {
		return " " + string(rune('0'+i))
	}
	return string(rune('0'+i/10)) + string(rune('0'+i%10))
}
