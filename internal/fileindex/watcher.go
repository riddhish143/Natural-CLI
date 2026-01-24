package fileindex

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	ix       *Index
	root     string
	inGitRepo bool
	watcher  *fsnotify.Watcher
	done     chan struct{}
	
	eventsMu sync.Mutex
	pending  map[string]fsnotify.Op
	debounce *time.Timer
}

func NewWatcher(ix *Index, root string, inGitRepo bool) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	fw := &Watcher{
		ix:        ix,
		root:      root,
		inGitRepo: inGitRepo,
		watcher:   w,
		done:      make(chan struct{}),
		pending:   make(map[string]fsnotify.Op),
	}

	if err := fw.addWatchesRecursive(root); err != nil {
		w.Close()
		return nil, err
	}

	return fw, nil
}

func (fw *Watcher) Start(ctx context.Context) {
	go fw.run(ctx)
}

func (fw *Watcher) Close() error {
	close(fw.done)
	return fw.watcher.Close()
}

func (fw *Watcher) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-fw.done:
			return
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}
			fw.handleEvent(event)
		case _, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
		}
	}
}

func (fw *Watcher) handleEvent(event fsnotify.Event) {
	if fw.shouldIgnore(event.Name) {
		return
	}

	fw.eventsMu.Lock()
	defer fw.eventsMu.Unlock()

	fw.pending[event.Name] = event.Op

	if fw.debounce != nil {
		fw.debounce.Stop()
	}
	fw.debounce = time.AfterFunc(200*time.Millisecond, fw.processPendingEvents)
}

func (fw *Watcher) processPendingEvents() {
	fw.eventsMu.Lock()
	events := fw.pending
	fw.pending = make(map[string]fsnotify.Op)
	fw.eventsMu.Unlock()

	if len(events) > 100 {
		fw.ix.Refresh(fw.root, fw.inGitRepo)
		fw.addWatchesRecursive(fw.root)
		return
	}

	for path, op := range events {
		relPath, err := filepath.Rel(fw.root, path)
		if err != nil {
			continue
		}

		if op&fsnotify.Remove != 0 || op&fsnotify.Rename != 0 {
			fw.ix.Remove(relPath)
			fw.watcher.Remove(path)
		}

		if op&fsnotify.Create != 0 || op&fsnotify.Write != 0 {
			info, err := os.Lstat(path)
			if err != nil {
				continue
			}

			if info.IsDir() {
				fw.addWatchesRecursive(path)
				fw.ix.Upsert(relPath, info, true)
			} else {
				fw.ix.Upsert(relPath, info, false)
			}
		}
	}
}

func (fw *Watcher) shouldIgnore(path string) bool {
	skipDirs := []string{".git", "node_modules", ".venv", "venv", "__pycache__", ".idea", ".vscode", "dist", "build", "vendor"}
	
	for _, skip := range skipDirs {
		if strings.Contains(path, string(filepath.Separator)+skip+string(filepath.Separator)) ||
			strings.HasSuffix(path, string(filepath.Separator)+skip) ||
			filepath.Base(path) == skip {
			return true
		}
	}
	return false
}

func (fw *Watcher) addWatchesRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if !d.IsDir() {
			return nil
		}

		if fw.shouldIgnore(path) {
			return filepath.SkipDir
		}

		fw.watcher.Add(path)
		return nil
	})
}
