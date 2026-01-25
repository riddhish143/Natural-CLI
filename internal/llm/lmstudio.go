package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/riddhishganeshmahajan/nsh/internal/config"
	ctx "github.com/riddhishganeshmahajan/nsh/internal/context"
)

// LMStudioProvider implements the Provider interface for LM Studio (OpenAI-compatible API)
type LMStudioProvider struct {
	baseURL string
	model   string
	timeout time.Duration
}

func NewLMStudioProvider(cfg config.Config) (*LMStudioProvider, error) {
	// Config is the single source of truth (defaults + config file + env vars)
	return &LMStudioProvider{
		baseURL: cfg.LMStudio.BaseURL,
		model:   cfg.LMStudio.Model,
		timeout: time.Duration(cfg.LMStudio.TimeoutSeconds) * time.Second,
	}, nil
}

func (l *LMStudioProvider) Name() string {
	return "lmstudio"
}

func (l *LMStudioProvider) Generate(c context.Context, userIntent string, envCtx ctx.Context, historySummary string) (*Generated, error) {
	prompt := buildPrompt(userIntent, envCtx, historySummary)

	reqBody := map[string]any{
		"model": l.model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": getSystemPrompt(envCtx),
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.2,
		"max_tokens":  2048,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := l.baseURL + "/chat/completions"

	httpCtx, cancel := context.WithTimeout(c, l.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("LM Studio request failed (is it running?): %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LM Studio error (status %d): %s", resp.StatusCode, string(body))
	}

	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from model")
	}

	text := openAIResp.Choices[0].Message.Content
	text = CleanModelResponse(text)
	text = CleanJSONResponse(text)

	var generated Generated
	if err := json.Unmarshal([]byte(text), &generated); err != nil {
		return nil, fmt.Errorf("failed to parse model response as JSON: %w\nRaw: %s", err, text)
	}

	if generated.Mode == "" {
		if generated.Command != "" {
			generated.Mode = ModeCommand
		} else if generated.Message != "" {
			generated.Mode = ModeAnswer
		}
	}

	return &generated, nil
}

func (l *LMStudioProvider) GenerateWithToolResult(c context.Context, userIntent string, envCtx ctx.Context, toolName, toolResult string) (*Generated, error) {
	prompt := buildFollowUpPrompt(userIntent, envCtx, toolName, toolResult)

	reqBody := map[string]any{
		"model": l.model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are nsh, a helpful terminal AI assistant. Respond in JSON format.",
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.2,
		"max_tokens":  2048,
	}

	jsonBody, _ := json.Marshal(reqBody)
	url := l.baseURL + "/chat/completions"

	httpCtx, cancel := context.WithTimeout(c, l.timeout)
	defer cancel()

	req, _ := http.NewRequestWithContext(httpCtx, "POST", url, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	json.Unmarshal(body, &openAIResp)
	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("no response")
	}

	text := CleanModelResponse(openAIResp.Choices[0].Message.Content)
	text = CleanJSONResponse(text)
	var generated Generated
	json.Unmarshal([]byte(text), &generated)

	return &generated, nil
}

func (l *LMStudioProvider) GenerateWithCommandError(c context.Context, userIntent string, envCtx ctx.Context, historySummary, failedCommand string, exitCode int, output string) (*Generated, error) {
	prompt := buildCommandRetryPrompt(userIntent, envCtx, historySummary, failedCommand, exitCode, output)

	reqBody := map[string]any{
		"model": l.model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are nsh, a terminal AI assistant. Analyze command failures and provide corrected commands. Respond in JSON format.",
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.2,
		"max_tokens":  2048,
	}

	jsonBody, _ := json.Marshal(reqBody)
	url := l.baseURL + "/chat/completions"

	httpCtx, cancel := context.WithTimeout(c, l.timeout)
	defer cancel()

	req, _ := http.NewRequestWithContext(httpCtx, "POST", url, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	json.Unmarshal(body, &openAIResp)
	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("no response")
	}

	text := CleanModelResponse(openAIResp.Choices[0].Message.Content)
	text = CleanJSONResponse(text)
	var generated Generated
	if err := json.Unmarshal([]byte(text), &generated); err != nil {
		return nil, err
	}

	if generated.Mode == "" {
		if generated.Command != "" {
			generated.Mode = ModeCommand
		} else if generated.Message != "" {
			generated.Mode = ModeAnswer
		}
	}

	return &generated, nil
}

func getSystemPrompt(envCtx ctx.Context) string {
	osNote := ""
	if envCtx.OS == "darwin" {
		osNote = `
IMPORTANT: This is macOS. Use BSD-compatible commands:
- Use 'find . -type f -exec ls -la {} +' instead of 'find -printf'
- Use 'du -sh' for disk usage
- Use 'stat -f' instead of 'stat -c'
- Use 'lsof' for port/process info`
	}

	fileIndexNote := ""
	if envCtx.FileIndex != nil {
		files, _ := envCtx.FileIndex.Count()
		fileIndexNote = fmt.Sprintf("\nWorkspace has %d indexed files. Check the file list in the prompt to answer file existence questions.", files)
	}

	return fmt.Sprintf(`You are nsh, an intelligent terminal AI assistant for %s/%s.
Shell: %s%s%s

Your job is to help users by:
1. Answering questions about commands, tools, and concepts
2. Generating REAL shell commands from natural language
3. Answering file existence queries using the workspace file index

CRITICAL: The "command" field MUST contain REAL executable shell commands like:
ls, find, cat, grep, awk, sed, curl, git, docker, npm, python, etc.

NEVER put these internal tool names in the "command" field:
file_list, file_search, file_read, content_search, diagnostics, process_info, port_info, package_search, package_info, web_search, web_fetch, git_status, git_log, git_diff, env_info

EXAMPLES:
User: "list files" -> {"mode":"command","command":"ls -la","execution":"auto"}
User: "find python files" -> {"mode":"command","command":"find . -name '*.py'","execution":"auto"}
User: "delete old logs" -> {"mode":"command","command":"rm *.log","execution":"confirm","execution_reason":"file deletion"}
WRONG: {"command":"file_list ."} <- NEVER DO THIS

Respond with valid JSON:
{"mode":"answer|command","message":"explanation","command":"REAL shell command","explanation":"what it does","execution":"auto|confirm","execution_reason":"why","confidence":0.9}

EXECUTION POLICY:
- "execution":"auto" - ONLY for read-only, safe commands (ls, cat, find, grep, ps, pwd, etc.)
- "execution":"confirm" - For file modifications, installations, sudo, network writes, or anything with side effects

Rules:
- For "is file X present" or "do we have file X": Check the file list and use mode="answer"
- For questions like "what is X", use mode="answer"
- For command requests, use mode="command" with REAL shell commands
- Always set "execution" for commands. When in doubt, use "confirm"
- Generate commands compatible with %s
- Be concise`, envCtx.OS, envCtx.Arch, envCtx.ShellName, osNote, fileIndexNote, envCtx.OS)
}
