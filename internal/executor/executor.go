package executor

import (
	"bytes"
	"io"
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

	// Stream output to UI immediately while still capturing for history
	var stdout, stderr bytes.Buffer
	cmd.Stdout = io.MultiWriter(&stdout, os.Stdout)
	cmd.Stderr = io.MultiWriter(&stderr, os.Stderr)

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

	// Output already streamed; only show a box if the command failed with no output.
	if output == "" && exitCode != 0 {
		ui.ShowOutputWithCode(output, exitCode)
	}

	return Result{ExitCode: exitCode, Output: output, Error: execErr}
}
