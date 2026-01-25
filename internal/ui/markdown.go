package ui

import (
	"strings"

	"github.com/charmbracelet/glamour"
)

var mdRenderer *glamour.TermRenderer

func initMarkdownRenderer(width int) {
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		mdRenderer = nil
		return
	}
	mdRenderer = r
}

func renderMarkdown(md string, width int) string {
	if width <= 0 {
		width = 80
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return stripMarkdown(md)
	}

	out, err := r.Render(md)
	if err != nil {
		return stripMarkdown(md)
	}

	return strings.TrimSpace(out)
}

func stripMarkdown(md string) string {
	md = strings.ReplaceAll(md, "**", "")
	md = strings.ReplaceAll(md, "__", "")
	md = strings.ReplaceAll(md, "*", "")
	md = strings.ReplaceAll(md, "_", "")
	md = strings.ReplaceAll(md, "`", "")
	md = strings.ReplaceAll(md, "###", "")
	md = strings.ReplaceAll(md, "##", "")
	md = strings.ReplaceAll(md, "#", "")
	return md
}
