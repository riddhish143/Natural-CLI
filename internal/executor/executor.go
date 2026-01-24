package executor

import (
	"bytes"
	"os"
	"os/exec"

	"github.com/riddhishganeshmahajan/nsh/internal/config"
	ctx "github.com/riddhishganeshmahajan/nsh/internal/context"
	"github.com/riddhishganeshmahajan/nsh/internal/ui"
)

type Result struct {
	ExitCode int
	Output   string
	Error    error
}

func Execute(command string, envCtx ctx.Context, cfg config.Config) Result {
	if cfg.Exec.DryRun {
		ui.ShowOutput("[DRY RUN] Would execute: " + command)
		return Result{ExitCode: 0}
	}

	shellArgs := []string{"-c", command}
	if cfg.Exec.UseLoginShell {
		shellArgs = []string{"-l", "-c", command}
	}

	cmd := exec.Command(envCtx.ShellPath, shellArgs...)
	cmd.Dir = envCtx.CWD
	cmd.Stdin = os.Stdin

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmd.Env = os.Environ()

	err := cmd.Run()
	
	// Combine output
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	// Determine exit code first
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

	// Display output in styled box with appropriate border color
	// Show box even when empty if command failed
	if output != "" || exitCode != 0 {
		ui.ShowOutputWithCode(output, exitCode)
	}

	return Result{ExitCode: exitCode, Output: output, Error: execErr}
}
