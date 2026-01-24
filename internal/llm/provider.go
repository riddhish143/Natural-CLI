package llm

import (
	"context"

	ctx "github.com/riddhishganeshmahajan/nsh/internal/context"
)

// Provider interface for LLM backends
type Provider interface {
	Generate(c context.Context, userIntent string, envCtx ctx.Context, historySummary string) (*Generated, error)
	GenerateWithToolResult(c context.Context, userIntent string, envCtx ctx.Context, toolName, toolResult string) (*Generated, error)
	GenerateWithCommandError(c context.Context, userIntent string, envCtx ctx.Context, historySummary, failedCommand string, exitCode int, output string) (*Generated, error)
	Name() string
}
