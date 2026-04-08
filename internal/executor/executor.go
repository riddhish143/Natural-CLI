package executor

import (
	"bytes"
	"os"
	"os/exec"

	"github.com/riddhishganeshmahajan/nsh/internal/config"
	ctx "github.com/riddhishganeshmahajan/nsh/internal/context"
	"github.com/riddhishganeshmahajan/nsh/internal/safety"
	"github.com/riddhishganeshmahajan/nsh/internal/ui"
)

type Result struct {
	ExitCode int
	Output   string
	Error    error
}

func Execute(command string, envCtx ctx.Context, cfg config.Config) Result {
	shellArgs := []string{"-c", command}
	if cfg.Exec.UseLoginShell {
		shellArgs = []string{"-l", "-c", command}
	}

	cmd := exec.Command(envCtx.ShellPath, shellArgs...)
	cmd.Dir = envCtx.CWD
	cmd.Stdin = os.Stdin

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = os.Environ()

	err := cmd.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	exitCode := 0
	var execErr error
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
			execErr = err
		}
	}

	if output != "" || exitCode != 0 {
		ui.ShowOutputWithCode(output, exitCode)
	}

	return Result{ExitCode: exitCode, Output: output, Error: execErr}
}

func ExecuteWithConfirmation(command string, envCtx ctx.Context, cfg config.Config, result safety.SafetyResult, needsConfirm bool) Result {
	if cfg.Exec.DryRun {
		ui.ShowDryRunCommand(command, result)
		return Result{ExitCode: 0}
	}

	if needsConfirm {
		if !ui.PromptConfirmation(command, result) {
			ui.ShowWarning("Command cancelled.")
			return Result{ExitCode: 130}
		}
	}

	return Execute(command, envCtx, cfg)
}