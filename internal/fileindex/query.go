package fileindex

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type QueryResult struct {
	Found   bool
	Entries []Entry
	Message string
}

func (ix *Index) Query(path string) QueryResult {
	normalized := normalizeRelPath(path)

	if entry, ok := ix.Get(normalized); ok {
		return QueryResult{
			Found:   true,
			Entries: []Entry{entry},
			Message: formatEntryDetails(entry),
		}
	}

	name := filepath.Base(path)
	if name != path && name != "." && name != "" {
		entries := ix.FindByName(name)
		if len(entries) > 0 {
			return QueryResult{
				Found:   true,
				Entries: entries,
				Message: formatMultipleMatches(name, entries),
			}
		}
	}

	entries := ix.FindByName(path)
	if len(entries) > 0 {
		return QueryResult{
			Found:   true,
			Entries: entries,
			Message: formatMultipleMatches(path, entries),
		}
	}

	return QueryResult{
		Found:   false,
		Message: fmt.Sprintf("'%s' not found in current workspace index", path),
	}
}

func (ix *Index) QueryExists(path string) (exists bool, details string) {
	result := ix.Query(path)
	if !result.Found {
		return false, result.Message
	}

	if len(result.Entries) == 1 {
		e := result.Entries[0]
		return true, fmt.Sprintf("Present at: %s (size: %s, modified: %s)",
			e.RelativePath, formatSize(e.SizeBytes), formatUnixTime(e.LastModified))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d matches:\n", len(result.Entries)))
	for _, e := range result.Entries {
		sb.WriteString(fmt.Sprintf("  - %s (size: %s)\n", e.RelativePath, formatSize(e.SizeBytes)))
	}
	return true, sb.String()
}

func (ix *Index) Summary() string {
	files, dirs := ix.Count()
	return fmt.Sprintf("Workspace: %s\nFiles: %d, Directories: %d\nIndexed at: %s",
		ix.Root, files, dirs, formatUnixTime(ix.GetBuiltAt()))
}

func formatEntryDetails(e Entry) string {
	typeStr := "file"
	if e.Type == TypeDir {
		typeStr = "directory"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Path: %s\n", e.RelativePath))
	sb.WriteString(fmt.Sprintf("Type: %s\n", typeStr))

	if e.Type == TypeFile {
		sb.WriteString(fmt.Sprintf("Size: %s (%d bytes)\n", formatSize(e.SizeBytes), e.SizeBytes))
		if e.Language != "" {
			sb.WriteString(fmt.Sprintf("Language: %s\n", e.Language))
		}
	}

	sb.WriteString(fmt.Sprintf("Modified: %s", formatUnixTime(e.LastModified)))

	return sb.String()
}

func formatMultipleMatches(query string, entries []Entry) string {
	if len(entries) == 1 {
		return formatEntryDetails(entries[0])
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d matches for '%s':\n", len(entries), query))
	for _, e := range entries {
		typeStr := "F"
		if e.Type == TypeDir {
			typeStr = "D"
		}
		sb.WriteString(fmt.Sprintf("  [%s] %s", typeStr, e.RelativePath))
		if e.Type == TypeFile {
			sb.WriteString(fmt.Sprintf(" (%s)", formatSize(e.SizeBytes)))
		}
		sb.WriteString("\n")
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func formatUnixTime(unix int64) string {
	if unix == 0 {
		return "unknown"
	}
	t := time.Unix(unix, 0)
	return t.Format("Jan 2, 2006 15:04")
}
