package fileindex

import (
	"time"
)

const (
	staleTTLSeconds = 300
)

func (ix *Index) Refresh(root string, inGitRepo bool) error {
	newIx, err := Build(root, inGitRepo)
	if err != nil {
		return err
	}

	ix.mu.Lock()
	defer ix.mu.Unlock()

	ix.Root = newIx.Root
	ix.entries = newIx.entries
	ix.byPath = newIx.byPath
	ix.byNameLower = newIx.byNameLower
	ix.gitHEAD = newIx.gitHEAD
	ix.builtAt = newIx.builtAt

	return nil
}

func (ix *Index) RefreshIfStale(root string, inGitRepo bool, currentGitHEAD string) (refreshed bool, err error) {
	ix.mu.RLock()
	needsRefresh := false

	if ix.Root != root {
		needsRefresh = true
	}

	if currentGitHEAD != "" && ix.gitHEAD != currentGitHEAD {
		needsRefresh = true
	}

	if time.Now().Unix()-ix.builtAt > staleTTLSeconds {
		needsRefresh = true
	}

	if len(ix.entries) == 0 {
		needsRefresh = true
	}

	ix.mu.RUnlock()

	if !needsRefresh {
		return false, nil
	}

	return true, ix.Refresh(root, inGitRepo)
}

func (ix *Index) IsStale(root string, currentGitHEAD string) bool {
	ix.mu.RLock()
	defer ix.mu.RUnlock()

	if ix.Root != root {
		return true
	}

	if currentGitHEAD != "" && ix.gitHEAD != currentGitHEAD {
		return true
	}

	if time.Now().Unix()-ix.builtAt > staleTTLSeconds {
		return true
	}

	if len(ix.entries) == 0 {
		return true
	}

	return false
}
