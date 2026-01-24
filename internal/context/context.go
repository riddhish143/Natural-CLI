package context

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/riddhishganeshmahajan/nsh/internal/fileindex"
)

type Context struct {
	CWD          string
	OS           string
	Arch         string
	ShellPath    string
	ShellName    string
	InGitRepo    bool
	GitRoot      string
	GitBranch    string
	GitHead      string
	Project      ProjectInfo
	FileIndex    *fileindex.Index
	fileWatcher  *fileindex.Watcher
	watcherCtx   context.Context
	watcherCancel context.CancelFunc
}

type ProjectInfo struct {
	Type       string
	Indicators []string
}

func Collect() Context {
	ctx := Context{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	ctx.CWD, _ = os.Getwd()
	ctx.ShellPath = os.Getenv("SHELL")
	if ctx.ShellPath != "" {
		ctx.ShellName = filepath.Base(ctx.ShellPath)
	} else {
		ctx.ShellName = "sh"
		ctx.ShellPath = "/bin/sh"
	}

	ctx.InGitRepo, ctx.GitRoot, ctx.GitBranch, ctx.GitHead = detectGit()
	ctx.Project = detectProject(ctx.CWD)

	return ctx
}

func (c *Context) WorkspaceRoot() string {
	if c.InGitRepo && c.GitRoot != "" {
		return c.GitRoot
	}
	return c.CWD
}

func (c *Context) EnsureFileIndex() error {
	root := c.WorkspaceRoot()

	if c.FileIndex == nil {
		ix, err := fileindex.Build(root, c.InGitRepo)
		if err != nil {
			return err
		}
		c.FileIndex = ix
		return nil
	}

	_, err := c.FileIndex.RefreshIfStale(root, c.InGitRepo, c.GitHead)
	return err
}

func (c *Context) RefreshFileIndex() error {
	root := c.WorkspaceRoot()

	if c.FileIndex == nil {
		ix, err := fileindex.Build(root, c.InGitRepo)
		if err != nil {
			return err
		}
		c.FileIndex = ix
		return nil
	}

	return c.FileIndex.Refresh(root, c.InGitRepo)
}

func detectGit() (inRepo bool, root, branch, head string) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return false, "", "", ""
	}
	root = strings.TrimSpace(string(out))

	cmd = exec.Command("git", "branch", "--show-current")
	out, err = cmd.Output()
	if err == nil {
		branch = strings.TrimSpace(string(out))
	}

	cmd = exec.Command("git", "rev-parse", "HEAD")
	out, err = cmd.Output()
	if err == nil {
		head = strings.TrimSpace(string(out))
	}

	return true, root, branch, head
}

func detectProject(cwd string) ProjectInfo {
	indicators := map[string]string{
		"package.json":      "node",
		"pyproject.toml":    "python",
		"requirements.txt":  "python",
		"go.mod":            "go",
		"Cargo.toml":        "rust",
		"pom.xml":           "java",
		"build.gradle":      "java",
		"Gemfile":           "ruby",
		"composer.json":     "php",
		"mix.exs":           "elixir",
		"Makefile":          "make",
		"CMakeLists.txt":    "cmake",
	}

	info := ProjectInfo{Type: "unknown"}

	for file, projectType := range indicators {
		path := filepath.Join(cwd, file)
		if _, err := os.Stat(path); err == nil {
			info.Type = projectType
			info.Indicators = append(info.Indicators, file)
		}
	}

	return info
}

func (c Context) Summary() string {
	var sb strings.Builder

	sb.WriteString("OS: " + c.OS + "/" + c.Arch + "\n")
	sb.WriteString("Shell: " + c.ShellName + "\n")
	sb.WriteString("CWD: " + c.CWD + "\n")

	if c.InGitRepo {
		sb.WriteString("Git: yes (branch: " + c.GitBranch + ")\n")
	} else {
		sb.WriteString("Git: no\n")
	}

	if c.Project.Type != "unknown" {
		sb.WriteString("Project: " + c.Project.Type + " (" + strings.Join(c.Project.Indicators, ", ") + ")\n")
	}

	if c.FileIndex != nil {
		files, dirs := c.FileIndex.Count()
		sb.WriteString("Workspace Index: " + c.WorkspaceRoot() + "\n")
		sb.WriteString("  Files: " + itoa(files) + ", Dirs: " + itoa(dirs) + "\n")
	}

	return sb.String()
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}

func (c *Context) FileExists(path string) (bool, string) {
	if c.FileIndex == nil {
		if err := c.EnsureFileIndex(); err != nil {
			return false, "Error building file index: " + err.Error()
		}
	}
	return c.FileIndex.QueryExists(path)
}

func (c *Context) FindFile(name string) []fileindex.Entry {
	if c.FileIndex == nil {
		if err := c.EnsureFileIndex(); err != nil {
			return nil
		}
	}
	return c.FileIndex.FindByName(name)
}

func (c *Context) FileIndexSummary() string {
	if c.FileIndex == nil {
		return "File index not initialized"
	}
	return c.FileIndex.Summary()
}

func (c *Context) StartFileWatcher() error {
	if c.fileWatcher != nil {
		return nil
	}

	if c.FileIndex == nil {
		if err := c.EnsureFileIndex(); err != nil {
			return err
		}
	}

	root := c.WorkspaceRoot()
	watcher, err := fileindex.NewWatcher(c.FileIndex, root, c.InGitRepo)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.watcherCtx = ctx
	c.watcherCancel = cancel
	c.fileWatcher = watcher

	watcher.Start(ctx)
	return nil
}

func (c *Context) StopFileWatcher() {
	if c.fileWatcher != nil {
		if c.watcherCancel != nil {
			c.watcherCancel()
		}
		c.fileWatcher.Close()
		c.fileWatcher = nil
	}
}

func (c *Context) IsWatching() bool {
	return c.fileWatcher != nil
}
