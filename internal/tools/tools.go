package tools

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/riddhishganeshmahajan/nsh/internal/fileindex"
)

type ToolResult struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

func FileSearch(pattern, root string, maxResults int) ToolResult {
	if root == "" {
		root = "."
	}
	if maxResults <= 0 {
		maxResults = 50
	}

	var matches []string
	count := 0

	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || count >= maxResults {
			return filepath.SkipDir
		}

		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" || name == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}

		matched, _ := filepath.Match(pattern, d.Name())
		if matched || strings.Contains(d.Name(), pattern) {
			matches = append(matches, path)
			count++
		}
		return nil
	})

	if len(matches) == 0 {
		return ToolResult{Success: true, Output: "No files found matching: " + pattern}
	}

	return ToolResult{Success: true, Output: strings.Join(matches, "\n")}
}

func FileList(path string, showHidden bool) ToolResult {
	if path == "" {
		path = "."
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return ToolResult{Success: false, Error: err.Error()}
	}

	var lines []string
	for _, e := range entries {
		name := e.Name()
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}

		info, _ := e.Info()
		suffix := ""
		size := ""
		if e.IsDir() {
			suffix = "/"
		} else if info != nil {
			size = formatSize(info.Size())
		}

		lines = append(lines, fmt.Sprintf("%s%s  %s", name, suffix, size))
	}

	return ToolResult{Success: true, Output: strings.Join(lines, "\n")}
}

func FileRead(path string, maxLines int) ToolResult {
	if isSensitivePath(path) {
		return ToolResult{Success: false, Error: "Cannot read sensitive file: " + path}
	}

	if maxLines <= 0 {
		maxLines = 100
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return ToolResult{Success: false, Error: err.Error()}
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines = append(lines, fmt.Sprintf("... (%d more lines)", len(lines)-maxLines))
	}

	return ToolResult{Success: true, Output: strings.Join(lines, "\n")}
}

func ContentSearch(pattern, root string, maxResults int) ToolResult {
	if root == "" {
		root = "."
	}
	if maxResults <= 0 {
		maxResults = 20
	}

	if _, err := exec.LookPath("rg"); err == nil {
		cmd := exec.Command("rg", "--no-heading", "--line-number", "--max-count", "3",
			"--max-filesize", "1M", "-m", strconv.Itoa(maxResults), pattern, root)
		out, _ := cmd.Output()
		if len(out) == 0 {
			return ToolResult{Success: true, Output: "No matches found for: " + pattern}
		}
		return ToolResult{Success: true, Output: string(out)}
	}

	if _, err := exec.LookPath("grep"); err == nil {
		cmd := exec.Command("grep", "-rn", "--include=*", "-m", strconv.Itoa(maxResults), pattern, root)
		out, _ := cmd.Output()
		if len(out) == 0 {
			return ToolResult{Success: true, Output: "No matches found for: " + pattern}
		}
		return ToolResult{Success: true, Output: string(out)}
	}

	return ToolResult{Success: false, Error: "Neither rg nor grep available"}
}

func SystemDiagnostics() ToolResult {
	var sb strings.Builder

	sb.WriteString("=== System Diagnostics ===\n\n")

	sb.WriteString("## CPU & Memory\n")
	if runtime.GOOS == "darwin" {
		if out, err := exec.Command("top", "-l", "1", "-n", "0").Output(); err == nil {
			lines := strings.Split(string(out), "\n")
			for i, line := range lines {
				if i < 10 && (strings.Contains(line, "CPU") || strings.Contains(line, "Mem") || strings.Contains(line, "Load")) {
					sb.WriteString(line + "\n")
				}
			}
		}
	} else {
		if out, err := exec.Command("free", "-h").Output(); err == nil {
			sb.WriteString(string(out))
		}
		if out, err := exec.Command("uptime").Output(); err == nil {
			sb.WriteString(string(out))
		}
	}

	sb.WriteString("\n## Disk Usage\n")
	if out, err := exec.Command("df", "-h").Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		for i, line := range lines {
			if i < 6 {
				sb.WriteString(line + "\n")
			}
		}
	}

	sb.WriteString("\n## Top Processes\n")
	if runtime.GOOS == "darwin" {
		if out, err := exec.Command("ps", "aux", "-r").Output(); err == nil {
			lines := strings.Split(string(out), "\n")
			for i, line := range lines {
				if i < 8 {
					sb.WriteString(line + "\n")
				}
			}
		}
	} else {
		if out, err := exec.Command("ps", "aux", "--sort=-%mem").Output(); err == nil {
			lines := strings.Split(string(out), "\n")
			for i, line := range lines {
				if i < 8 {
					sb.WriteString(line + "\n")
				}
			}
		}
	}

	sb.WriteString("\n## Network Listeners\n")
	if runtime.GOOS == "darwin" {
		if out, err := exec.Command("lsof", "-iTCP", "-sTCP:LISTEN", "-P", "-n").Output(); err == nil {
			lines := strings.Split(string(out), "\n")
			for i, line := range lines {
				if i < 15 {
					sb.WriteString(line + "\n")
				}
			}
		}
	} else {
		if out, err := exec.Command("ss", "-tlnp").Output(); err == nil {
			sb.WriteString(string(out))
		}
	}

	return ToolResult{Success: true, Output: sb.String()}
}

func ProcessInfo(query string) ToolResult {
	var cmd *exec.Cmd

	if runtime.GOOS == "darwin" {
		cmd = exec.Command("ps", "aux")
	} else {
		cmd = exec.Command("ps", "aux")
	}

	out, err := cmd.Output()
	if err != nil {
		return ToolResult{Success: false, Error: err.Error()}
	}

	lines := strings.Split(string(out), "\n")
	var matches []string
	matches = append(matches, lines[0])

	query = strings.ToLower(query)
	for _, line := range lines[1:] {
		if strings.Contains(strings.ToLower(line), query) {
			matches = append(matches, line)
		}
	}

	if len(matches) <= 1 {
		return ToolResult{Success: true, Output: "No processes found matching: " + query}
	}

	return ToolResult{Success: true, Output: strings.Join(matches, "\n")}
}

func PortInfo(port string) ToolResult {
	var cmd *exec.Cmd

	if runtime.GOOS == "darwin" {
		cmd = exec.Command("lsof", "-i", ":"+port, "-P", "-n")
	} else {
		cmd = exec.Command("ss", "-tlnp", "sport", "=", ":"+port)
	}

	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return ToolResult{Success: true, Output: "Nothing is using port " + port}
	}

	return ToolResult{Success: true, Output: string(out)}
}

func PackageSearch(query string) ToolResult {
	mgr := detectPackageManager()
	if mgr == "" {
		return ToolResult{Success: false, Error: "No supported package manager found"}
	}

	var cmd *exec.Cmd
	switch mgr {
	case "brew":
		cmd = exec.Command("brew", "search", query)
	case "apt":
		cmd = exec.Command("apt-cache", "search", query)
	case "dnf", "yum":
		cmd = exec.Command(mgr, "search", query)
	case "pacman":
		cmd = exec.Command("pacman", "-Ss", query)
	default:
		return ToolResult{Success: false, Error: "Package search not implemented for: " + mgr}
	}

	out, err := cmd.Output()
	if err != nil {
		return ToolResult{Success: false, Error: err.Error()}
	}

	lines := strings.Split(string(out), "\n")
	if len(lines) > 20 {
		lines = lines[:20]
		lines = append(lines, "... (more results)")
	}

	return ToolResult{Success: true, Output: strings.Join(lines, "\n")}
}

func PackageInfo(name string) ToolResult {
	mgr := detectPackageManager()
	if mgr == "" {
		return ToolResult{Success: false, Error: "No supported package manager found"}
	}

	var cmd *exec.Cmd
	switch mgr {
	case "brew":
		cmd = exec.Command("brew", "info", name)
	case "apt":
		cmd = exec.Command("apt-cache", "show", name)
	case "dnf", "yum":
		cmd = exec.Command(mgr, "info", name)
	case "pacman":
		cmd = exec.Command("pacman", "-Si", name)
	default:
		return ToolResult{Success: false, Error: "Package info not implemented for: " + mgr}
	}

	out, err := cmd.Output()
	if err != nil {
		return ToolResult{Success: false, Error: "Package not found: " + name}
	}

	return ToolResult{Success: true, Output: string(out)}
}

func WebSearch(query string) ToolResult {
	url := "https://html.duckduckgo.com/html/?q=" + strings.ReplaceAll(query, " ", "+")

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; nsh/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return ToolResult{Success: false, Error: "Search failed: " + err.Error()}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	results := parseSearchResults(html)
	if len(results) == 0 {
		return ToolResult{Success: true, Output: "No search results found for: " + query}
	}

	return ToolResult{Success: true, Output: strings.Join(results, "\n\n")}
}

func WebFetch(url string, maxBytes int) ToolResult {
	if maxBytes <= 0 {
		maxBytes = 5000
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; nsh/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return ToolResult{Success: false, Error: "Fetch failed: " + err.Error()}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, int64(maxBytes)))
	text := stripHTML(string(body))

	if len(text) > maxBytes {
		text = text[:maxBytes] + "..."
	}

	return ToolResult{Success: true, Output: text}
}

func GitStatus() ToolResult {
	out, err := exec.Command("git", "status", "--short", "--branch").Output()
	if err != nil {
		return ToolResult{Success: false, Error: "Not a git repository or git not installed"}
	}
	return ToolResult{Success: true, Output: string(out)}
}

func GitLog(count int) ToolResult {
	if count <= 0 {
		count = 10
	}
	out, err := exec.Command("git", "log", "--oneline", "-n", strconv.Itoa(count)).Output()
	if err != nil {
		return ToolResult{Success: false, Error: err.Error()}
	}
	return ToolResult{Success: true, Output: string(out)}
}

func GitDiff() ToolResult {
	out, err := exec.Command("git", "diff", "--stat").Output()
	if err != nil {
		return ToolResult{Success: false, Error: err.Error()}
	}
	if len(out) == 0 {
		return ToolResult{Success: true, Output: "No uncommitted changes"}
	}
	return ToolResult{Success: true, Output: string(out)}
}

func EnvInfo(pattern string) ToolResult {
	var matches []string
	pattern = strings.ToLower(pattern)

	for _, env := range os.Environ() {
		if isSensitiveEnv(env) {
			continue
		}
		if pattern == "" || strings.Contains(strings.ToLower(env), pattern) {
			matches = append(matches, env)
		}
	}

	sort.Strings(matches)
	if len(matches) > 50 {
		matches = matches[:50]
	}

	return ToolResult{Success: true, Output: strings.Join(matches, "\n")}
}

func detectPackageManager() string {
	managers := []string{"brew", "apt", "dnf", "yum", "pacman", "apk", "zypper"}
	for _, mgr := range managers {
		if _, err := exec.LookPath(mgr); err == nil {
			return mgr
		}
	}
	return ""
}

func isSensitivePath(path string) bool {
	sensitive := []string{
		".env", ".pem", ".key", "id_rsa", "id_ed25519", "id_dsa",
		".ssh/", "credentials", "secrets", ".aws/", ".netrc",
		"password", "token", ".gnupg/",
	}
	lower := strings.ToLower(path)
	for _, s := range sensitive {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}

func isSensitiveEnv(env string) bool {
	sensitive := []string{"KEY", "SECRET", "TOKEN", "PASSWORD", "CREDENTIAL", "AUTH"}
	upper := strings.ToUpper(env)
	for _, s := range sensitive {
		if strings.Contains(upper, s) {
			return true
		}
	}
	return false
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func parseSearchResults(html string) []string {
	var results []string

	re := regexp.MustCompile(`<a[^>]+class="result__a"[^>]*href="([^"]*)"[^>]*>([^<]*)</a>`)
	matches := re.FindAllStringSubmatch(html, 5)

	for _, m := range matches {
		if len(m) >= 3 {
			title := strings.TrimSpace(m[2])
			rawURL := m[1]
			if strings.HasPrefix(rawURL, "//duckduckgo.com/l/?uddg=") {
				rawURL = strings.TrimPrefix(rawURL, "//duckduckgo.com/l/?uddg=")
				if idx := strings.Index(rawURL, "&"); idx != -1 {
					rawURL = rawURL[:idx]
				}
			}
			// Decode URL-encoded characters
			decodedURL, err := url.QueryUnescape(rawURL)
			if err != nil {
				decodedURL = rawURL
			}
			results = append(results, fmt.Sprintf("• %s\n  %s", title, decodedURL))
		}
	}

	if len(results) == 0 {
		re = regexp.MustCompile(`<a[^>]+href="(https?://[^"]+)"[^>]*>([^<]{10,})</a>`)
		matches = re.FindAllStringSubmatch(html, 5)
		for _, m := range matches {
			if len(m) >= 3 && !strings.Contains(m[1], "duckduckgo") {
				decodedURL, err := url.QueryUnescape(m[1])
				if err != nil {
					decodedURL = m[1]
				}
				results = append(results, fmt.Sprintf("• %s\n  %s", strings.TrimSpace(m[2]), decodedURL))
			}
		}
	}

	return results
}

func stripHTML(html string) string {
	re := regexp.MustCompile(`<script[^>]*>[\s\S]*?</script>`)
	html = re.ReplaceAllString(html, "")
	re = regexp.MustCompile(`<style[^>]*>[\s\S]*?</style>`)
	html = re.ReplaceAllString(html, "")
	re = regexp.MustCompile(`<[^>]+>`)
	html = re.ReplaceAllString(html, " ")
	re = regexp.MustCompile(`\s+`)
	html = re.ReplaceAllString(html, " ")
	return strings.TrimSpace(html)
}

func RunCommand(command, shell string) ToolResult {
	if shell == "" {
		shell = os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
	}

	ctx, cancel := exec.Command(shell, "-c", command), func() {}
	_ = cancel

	var stdout, stderr bytes.Buffer
	ctx.Stdout = &stdout
	ctx.Stderr = &stderr

	err := ctx.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n[stderr]: " + stderr.String()
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return ToolResult{
				Success: false,
				Output:  output,
				Error:   fmt.Sprintf("Exit code: %d", exitErr.ExitCode()),
			}
		}
		return ToolResult{Success: false, Output: output, Error: err.Error()}
	}

	return ToolResult{Success: true, Output: output}
}

func FileIndexQuery(path string, ix *fileindex.Index) ToolResult {
	if ix == nil {
		return ToolResult{Success: false, Error: "File index not initialized"}
	}

	result := ix.Query(path)
	return ToolResult{Success: result.Found, Output: result.Message}
}

func FileIndexExists(path string, ix *fileindex.Index) ToolResult {
	if ix == nil {
		return ToolResult{Success: false, Error: "File index not initialized"}
	}

	exists, details := ix.QueryExists(path)
	if exists {
		return ToolResult{Success: true, Output: "Present: " + details}
	}
	return ToolResult{Success: true, Output: details}
}

func FileIndexSummary(ix *fileindex.Index) ToolResult {
	if ix == nil {
		return ToolResult{Success: false, Error: "File index not initialized"}
	}

	return ToolResult{Success: true, Output: ix.Summary()}
}

func FileIndexFindByExt(ext string, ix *fileindex.Index) ToolResult {
	if ix == nil {
		return ToolResult{Success: false, Error: "File index not initialized"}
	}

	entries := ix.FindByExtension(ext)
	if len(entries) == 0 {
		return ToolResult{Success: true, Output: "No files found with extension: " + ext}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d files with extension .%s:\n", len(entries), ext))
	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("  %s (%s)\n", e.RelativePath, formatSize(e.SizeBytes)))
	}
	return ToolResult{Success: true, Output: sb.String()}
}
