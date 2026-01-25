package llm

import (
	"fmt"
	"strings"
)

// extractFirstJSON finds the first complete JSON object/array in s.
// It is nesting-aware and string-aware (so braces inside "..." don't count).
func extractFirstJSON(s string) (string, bool) {
	start := -1
	for i := 0; i < len(s); i++ {
		if s[i] == '{' || s[i] == '[' {
			start = i
			break
		}
	}
	if start == -1 {
		return "", false
	}

	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(s); i++ {
		c := s[i]

		if inString {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == '"' {
				inString = false
			}
			continue
		}

		switch c {
		case '"':
			inString = true
		case '{', '[':
			depth++
		case '}', ']':
			depth--
			if depth == 0 {
				return s[start : i+1], true
			}
		}
	}

	return "", false
}

// escapeControlCharsInStrings repairs invalid JSON produced by LLMs,
// specifically literal newlines/tabs/etc inside JSON string values.
func escapeControlCharsInStrings(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	inString := false
	escaped := false

	for i := 0; i < len(s); i++ {
		c := s[i]

		if inString {
			if escaped {
				escaped = false
				b.WriteByte(c)
				continue
			}
			if c == '\\' {
				escaped = true
				b.WriteByte(c)
				continue
			}
			if c == '"' {
				inString = false
				b.WriteByte(c)
				continue
			}

			switch c {
			case '\n':
				b.WriteString(`\n`)
			case '\r':
				b.WriteString(`\n`)
			case '\t':
				b.WriteString(`\t`)
			default:
				if c < 0x20 {
					fmt.Fprintf(&b, `\u%04x`, c)
				} else {
					b.WriteByte(c)
				}
			}
			continue
		}

		if c == '"' {
			inString = true
			b.WriteByte(c)
			continue
		}

		b.WriteByte(c)
	}

	return b.String()
}

// removeTrailingCommas removes trailing commas outside strings: {"a":1,} or [1,]
func removeTrailingCommas(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	inString := false
	escaped := false

	for i := 0; i < len(s); i++ {
		c := s[i]

		if inString {
			if escaped {
				escaped = false
				b.WriteByte(c)
				continue
			}
			if c == '\\' {
				escaped = true
				b.WriteByte(c)
				continue
			}
			if c == '"' {
				inString = false
			}
			b.WriteByte(c)
			continue
		}

		if c == '"' {
			inString = true
			b.WriteByte(c)
			continue
		}

		if c == ',' {
			j := i + 1
			for j < len(s) && (s[j] == ' ' || s[j] == '\n' || s[j] == '\r' || s[j] == '\t') {
				j++
			}
			if j < len(s) && (s[j] == '}' || s[j] == ']') {
				continue
			}
		}

		b.WriteByte(c)
	}

	return b.String()
}

// CleanModelResponse removes model-specific tokens and extracts JSON
func CleanModelResponse(text string) string {
	text = strings.TrimSpace(text)

	replacements := []string{
		"<|channel|>final <|constrain|>JSON<|message|>", "",
		"<|channel|>", "",
		"<|constrain|>", "",
		"<|message|>", "",
		"<|im_start|>", "",
		"<|im_end|>", "",
		"</s>", "",
		"<s>", "",
		"<|assistant|>", "",
		"<|user|>", "",
		"<|system|>", "",
	}
	text = strings.NewReplacer(replacements...).Replace(text)

	if idx := strings.Index(text, "```"); idx != -1 {
		rest := text[idx+3:]
		rest = strings.TrimPrefix(rest, "json")
		rest = strings.TrimPrefix(rest, "JSON")
		rest = strings.TrimSpace(rest)
		if end := strings.Index(rest, "```"); end != -1 {
			text = rest[:end]
		}
	}

	if js, ok := extractFirstJSON(text); ok {
		return strings.TrimSpace(js)
	}

	return strings.TrimSpace(text)
}

// CleanJSONResponse repairs common LLM JSON issues
func CleanJSONResponse(text string) string {
	text = strings.TrimSpace(text)

	if js, ok := extractFirstJSON(text); ok {
		text = js
	}

	text = escapeControlCharsInStrings(text)
	text = removeTrailingCommas(text)

	return strings.TrimSpace(text)
}
