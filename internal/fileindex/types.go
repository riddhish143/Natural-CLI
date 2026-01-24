package fileindex

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type EntryType uint8

const (
	TypeFile EntryType = iota
	TypeDir
)

type Entry struct {
	RelativePath string    // slash-normalized, relative to Root
	Name         string    // base name
	Extension    string    // without dot, lowercase
	Type         EntryType
	SizeBytes    int64
	LastModified int64  // unix seconds
	Hash         string // optional; empty until computed
	Language     string // optional; derived from Extension
}

type Index struct {
	Root string // absolute root indexed

	entries []Entry

	byPath      map[string]int   // relpath -> index into entries
	byNameLower map[string][]int // lowercase basename -> indices

	gitHEAD string
	builtAt int64

	mu sync.RWMutex
}

func NewIndex(root string) *Index {
	return &Index{
		Root:        root,
		entries:     make([]Entry, 0),
		byPath:      make(map[string]int),
		byNameLower: make(map[string][]int),
	}
}

func (ix *Index) addEntry(e Entry) {
	idx := len(ix.entries)
	ix.entries = append(ix.entries, e)
	ix.byPath[e.RelativePath] = idx
	nameLower := strings.ToLower(e.Name)
	ix.byNameLower[nameLower] = append(ix.byNameLower[nameLower], idx)
}

func (ix *Index) Upsert(relPath string, info os.FileInfo, isDir bool) {
	ix.mu.Lock()
	defer ix.mu.Unlock()

	normalized := normalizeRelPath(relPath)
	name := filepath.Base(relPath)
	ext := ""
	var entryType EntryType
	var size int64

	if isDir {
		entryType = TypeDir
	} else {
		entryType = TypeFile
		ext = strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
		size = info.Size()
	}

	entry := Entry{
		RelativePath: normalized,
		Name:         name,
		Extension:    ext,
		Type:         entryType,
		SizeBytes:    size,
		LastModified: info.ModTime().Unix(),
		Language:     languageFromExt(ext),
	}

	if existingIdx, exists := ix.byPath[normalized]; exists {
		ix.entries[existingIdx] = entry
	} else {
		ix.addEntryLocked(entry)
	}
}

func (ix *Index) addEntryLocked(e Entry) {
	idx := len(ix.entries)
	ix.entries = append(ix.entries, e)
	ix.byPath[e.RelativePath] = idx
	nameLower := strings.ToLower(e.Name)
	ix.byNameLower[nameLower] = append(ix.byNameLower[nameLower], idx)
}

func (ix *Index) Remove(relPath string) {
	ix.mu.Lock()
	defer ix.mu.Unlock()

	normalized := normalizeRelPath(relPath)
	
	idx, exists := ix.byPath[normalized]
	if !exists {
		return
	}

	entry := ix.entries[idx]
	delete(ix.byPath, normalized)

	nameLower := strings.ToLower(entry.Name)
	if indices, ok := ix.byNameLower[nameLower]; ok {
		newIndices := make([]int, 0, len(indices))
		for _, i := range indices {
			if i != idx {
				newIndices = append(newIndices, i)
			}
		}
		if len(newIndices) == 0 {
			delete(ix.byNameLower, nameLower)
		} else {
			ix.byNameLower[nameLower] = newIndices
		}
	}

	ix.entries[idx] = Entry{}

	if entry.Type == TypeDir {
		prefix := normalized + "/"
		for path, i := range ix.byPath {
			if strings.HasPrefix(path, prefix) {
				childEntry := ix.entries[i]
				delete(ix.byPath, path)
				childNameLower := strings.ToLower(childEntry.Name)
				if childIndices, ok := ix.byNameLower[childNameLower]; ok {
					newChildIndices := make([]int, 0, len(childIndices))
					for _, ci := range childIndices {
						if ci != i {
							newChildIndices = append(newChildIndices, ci)
						}
					}
					if len(newChildIndices) == 0 {
						delete(ix.byNameLower, childNameLower)
					} else {
						ix.byNameLower[childNameLower] = newChildIndices
					}
				}
				ix.entries[i] = Entry{}
			}
		}
	}
}

func (ix *Index) Exists(relPath string) bool {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	normalized := normalizeRelPath(relPath)
	_, ok := ix.byPath[normalized]
	return ok
}

func (ix *Index) Get(relPath string) (Entry, bool) {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	normalized := normalizeRelPath(relPath)
	if idx, ok := ix.byPath[normalized]; ok {
		return ix.entries[idx], true
	}
	return Entry{}, false
}

func (ix *Index) FindByName(name string) []Entry {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	nameLower := strings.ToLower(name)
	indices, ok := ix.byNameLower[nameLower]
	if !ok {
		return nil
	}
	results := make([]Entry, 0, len(indices))
	for _, idx := range indices {
		results = append(results, ix.entries[idx])
	}
	return results
}

func (ix *Index) FindByNameCaseSensitive(name string) []Entry {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	nameLower := strings.ToLower(name)
	indices, ok := ix.byNameLower[nameLower]
	if !ok {
		return nil
	}
	results := make([]Entry, 0)
	for _, idx := range indices {
		if ix.entries[idx].Name == name {
			results = append(results, ix.entries[idx])
		}
	}
	return results
}

func (ix *Index) FindByExtension(ext string) []Entry {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	results := make([]Entry, 0)
	for _, e := range ix.entries {
		if e.Extension == ext {
			results = append(results, e)
		}
	}
	return results
}

func (ix *Index) AllFiles() []Entry {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	results := make([]Entry, 0)
	for _, e := range ix.entries {
		if e.Type == TypeFile {
			results = append(results, e)
		}
	}
	return results
}

func (ix *Index) AllDirs() []Entry {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	results := make([]Entry, 0)
	for _, e := range ix.entries {
		if e.Type == TypeDir {
			results = append(results, e)
		}
	}
	return results
}

func (ix *Index) Count() (files int, dirs int) {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	for _, e := range ix.entries {
		if e.Type == TypeFile {
			files++
		} else {
			dirs++
		}
	}
	return
}

func (ix *Index) GetGitHEAD() string {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return ix.gitHEAD
}

func (ix *Index) GetBuiltAt() int64 {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return ix.builtAt
}

func normalizeRelPath(p string) string {
	p = filepath.ToSlash(p)
	p = strings.TrimPrefix(p, "./")
	p = strings.TrimPrefix(p, "/")
	return p
}

var extensionToLanguage = map[string]string{
	"go":    "go",
	"py":    "python",
	"js":    "javascript",
	"ts":    "typescript",
	"jsx":   "javascript",
	"tsx":   "typescript",
	"rb":    "ruby",
	"rs":    "rust",
	"java":  "java",
	"c":     "c",
	"cpp":   "cpp",
	"cc":    "cpp",
	"h":     "c",
	"hpp":   "cpp",
	"cs":    "csharp",
	"php":   "php",
	"swift": "swift",
	"kt":    "kotlin",
	"scala": "scala",
	"sh":    "shell",
	"bash":  "shell",
	"zsh":   "shell",
	"ps1":   "powershell",
	"sql":   "sql",
	"html":  "html",
	"css":   "css",
	"scss":  "scss",
	"sass":  "sass",
	"less":  "less",
	"json":  "json",
	"yaml":  "yaml",
	"yml":   "yaml",
	"xml":   "xml",
	"md":    "markdown",
	"txt":   "text",
	"toml":  "toml",
	"ini":   "ini",
	"cfg":   "config",
	"conf":  "config",
}

func languageFromExt(ext string) string {
	if lang, ok := extensionToLanguage[ext]; ok {
		return lang
	}
	return ""
}
