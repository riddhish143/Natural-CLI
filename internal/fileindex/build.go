package fileindex

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func Build(root string, inGitRepo bool) (*Index, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	ix := NewIndex(root)

	if inGitRepo {
		err = buildFromGit(ix, root)
	} else {
		err = buildFromWalk(ix, root)
	}

	if err != nil {
		return nil, err
	}

	ix.builtAt = time.Now().Unix()

	if inGitRepo {
		ix.gitHEAD = getGitHEAD(root)
	}

	return ix, nil
}

func buildFromGit(ix *Index, root string) error {
	dirs := make(map[string]struct{})

	trackedFiles, err := runGitLsFiles(root, false)
	if err != nil {
		return buildFromWalk(ix, root)
	}

	untrackedFiles, _ := runGitLsFiles(root, true)

	allFiles := append(trackedFiles, untrackedFiles...)

	for _, relPath := range allFiles {
		if relPath == "" {
			continue
		}

		absPath := filepath.Join(root, relPath)
		info, err := os.Lstat(absPath)
		if err != nil {
			continue
		}

		if info.IsDir() {
			continue
		}

		name := filepath.Base(relPath)
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))

		entry := Entry{
			RelativePath: normalizeRelPath(relPath),
			Name:         name,
			Extension:    ext,
			Type:         TypeFile,
			SizeBytes:    info.Size(),
			LastModified: info.ModTime().Unix(),
			Language:     languageFromExt(ext),
		}
		ix.addEntry(entry)

		dir := filepath.Dir(relPath)
		for dir != "." && dir != "" {
			dirs[dir] = struct{}{}
			dir = filepath.Dir(dir)
		}
	}

	for dir := range dirs {
		absPath := filepath.Join(root, dir)
		info, err := os.Lstat(absPath)
		modTime := int64(0)
		if err == nil {
			modTime = info.ModTime().Unix()
		}

		entry := Entry{
			RelativePath: normalizeRelPath(dir),
			Name:         filepath.Base(dir),
			Type:         TypeDir,
			LastModified: modTime,
		}
		ix.addEntry(entry)
	}

	return nil
}

func runGitLsFiles(root string, untracked bool) ([]string, error) {
	var args []string
	if untracked {
		args = []string{"ls-files", "-z", "-o", "--exclude-standard"}
	} else {
		args = []string{"ls-files", "-z"}
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var files []string
	for _, f := range bytes.Split(out, []byte{0}) {
		if len(f) > 0 {
			files = append(files, string(f))
		}
	}
	return files, nil
}

func buildFromWalk(ix *Index, root string) error {
	skipDirs := map[string]bool{
		".git":         true,
		"node_modules": true,
		".venv":        true,
		"venv":         true,
		"__pycache__":  true,
		".idea":        true,
		".vscode":      true,
		"dist":         true,
		"build":        true,
		"vendor":       true,
	}

	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() && skipDirs[d.Name()] {
			return filepath.SkipDir
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil || relPath == "." {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		name := d.Name()
		var entryType EntryType
		var ext string
		var size int64

		if d.IsDir() {
			entryType = TypeDir
		} else {
			entryType = TypeFile
			ext = strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
			size = info.Size()
		}

		entry := Entry{
			RelativePath: normalizeRelPath(relPath),
			Name:         name,
			Extension:    ext,
			Type:         entryType,
			SizeBytes:    size,
			LastModified: info.ModTime().Unix(),
			Language:     languageFromExt(ext),
		}
		ix.addEntry(entry)

		return nil
	})
}

func getGitHEAD(root string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
