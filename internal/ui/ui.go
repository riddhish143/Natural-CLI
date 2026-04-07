package ui

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/mattn/go-runewidth"
	"github.com/riddhishganeshmahajan/nsh/internal/config"
	"github.com/riddhishganeshmahajan/nsh/internal/llm"
	"github.com/riddhishganeshmahajan/nsh/internal/safety"
)

var (
	cyan    = color.New(color.FgHiCyan)
	magenta = color.New(color.FgHiMagenta)
	green   = color.New(color.FgHiGreen)
	yellow  = color.New(color.FgHiYellow)
	red     = color.New(color.FgHiRed)
	blue    = color.New(color.FgHiBlue)
	white   = color.New(color.FgHiWhite)
	dim     = color.New(color.Faint)
)

const (
	legacyBoxWidth    = 70
	legacyBoxMinHeight = 6
)

// Matches all ANSI escape sequences including colors, cursor movement, screen clearing, etc.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]|\x1b\][^\x07]*\x07|\x1b[()][AB012]`)

var activeSpinner *AnimatedSpinner

// promptActive tracks if ShowPrompt left cursor on an unfinished line
var promptActive bool

func ensureNewlineAfterPrompt() {
	if promptActive {
		fmt.Println()
		promptActive = false
	}
}

const baseIndent = "  "

var statusLineStyle = lipgloss.NewStyle().Foreground(subtleColor)

// PrintStatusLine prints a consistently indented status message
func PrintStatusLine(format string, args ...any) {
	ensureNewlineAfterPrompt()
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s\n", baseIndent, statusLineStyle.Render(msg))
}

// PrintBlock prints text with baseIndent applied to every line.
// Use this for multi-line summaries, history, tool outputs, etc.
func PrintBlock(text string) {
	ensureNewlineAfterPrompt()
	text = strings.TrimRight(text, "\n")
	if text == "" {
		fmt.Println(baseIndent)
		return
	}

	for _, line := range strings.Split(text, "\n") {
		fmt.Println(baseIndent + line)
	}
}

func ShowTranslating() {
	ensureNewlineAfterPrompt()
	activeSpinner = NewSpinner("Thinking...")
	activeSpinner.Start()
}

func ClearTranslating() {
	if activeSpinner != nil {
		activeSpinner.Stop()
		activeSpinner = nil
	}
}

func ShowAnswer(message string) {
	ensureNewlineAfterPrompt()
	refreshWidth()
	// Render markdown with inner width (box - border - padding)
	innerWidth := boxWidth - 2 - 4 // 2 for border, 4 for padding (2 each side)
	rendered := renderMarkdown(message, innerWidth)
	content := successTitleStyle.Render(successIcon+" Answer") + "\n\n" + rendered
	fmt.Println(answerStyle.Render(content))
}

func ShowClarify(message string) {
	ensureNewlineAfterPrompt()
	refreshWidth()
	innerWidth := boxWidth - 2 - 4
	rendered := renderMarkdown(message, innerWidth)
	content := warningTitleStyle.Render(warningIcon+" Clarification Needed") + "\n\n" + rendered
	fmt.Println(warningStyle.Render(content))
}

func ShowPlanStart(message string, stepCount int) {
	ensureNewlineAfterPrompt()
	refreshWidth()
	title := lipgloss.NewStyle().Bold(true).Foreground(infoColor).Render(fmt.Sprintf("◇ Plan (%d steps)", stepCount))
	content := title
	if message != "" {
		innerWidth := boxWidth - 2 - 4
		rendered := renderMarkdown(message, innerWidth)
		content += "\n\n" + rendered
	}
	fmt.Println(infoStyle.Render(content))
}

func ShowSearchResults(title, results string) {
	ensureNewlineAfterPrompt()
	refreshWidth()
	titleRendered := lipgloss.NewStyle().Bold(true).Foreground(infoColor).Render("🔍 " + title)
	content := titleRendered + "\n\n" + results
	
	// Use a white/text colored border for search results
	searchStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(textColor).
		Padding(1, 2).
		Margin(1, 2).
		Width(boxWidth)
	
	fmt.Println(searchStyle.Render(content))
}

func ShowToolOutput(title, output string) {
	ensureNewlineAfterPrompt()
	refreshWidth()
	titleRendered := lipgloss.NewStyle().Bold(true).Foreground(subtleColor).Render(title)
	content := titleRendered + "\n\n" + output
	fmt.Println(outputStyle.Render(content))
}

func ShowPlanStep(id, tool, purpose string) {
	magenta.Printf("  ⚡ %s", tool)
	dim.Printf(" • %s\n", purpose)
}

func ShowToolResult(tool, output string) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	maxLines := 6
	if len(lines) > maxLines {
		for i := 0; i < maxLines; i++ {
			dim.Printf("    %s\n", safeTruncate(lines[i], 65))
		}
		dim.Printf("    ... +%d more lines\n", len(lines)-maxLines)
	} else {
		for _, line := range lines {
			dim.Printf("    %s\n", safeTruncate(line, 65))
		}
	}
}

func ShowToolError(tool, err string) {
	red.Printf("  ✗ %s: %s\n", tool, err)
}

func ShowCommand(gen *llm.Generated, result safety.SafetyResult, cfg config.Config) {
	ensureNewlineAfterPrompt()
	refreshWidth()

	// Build content
	title := titleStyle.Render(commandIcon + " Command")
	cmd := commandTextStyle.Render(gen.Command)

	// Explanation
	explanation := gen.Explanation
	if explanation == "" {
		explanation = gen.Message
	}
	explText := ""
	if explanation != "" {
		explText = "\n" + explanationStyle.Render(explanation)
	}

	// Risk indicator
	riskIcon := successIcon
	riskStyle := lipgloss.NewStyle().Foreground(successColor)
	switch result.Risk {
	case safety.RiskMedium:
		riskIcon = warningIcon
		riskStyle = lipgloss.NewStyle().Foreground(warningColor)
	case safety.RiskHigh, safety.RiskBlocked:
		riskIcon = errorIcon
		riskStyle = lipgloss.NewStyle().Foreground(errorColor)
	}

	riskText := riskStyle.Render(fmt.Sprintf("%s %s", riskIcon, result.Risk))
	if gen.Confidence > 0 {
		riskText += lipgloss.NewStyle().Foreground(subtleColor).Render(fmt.Sprintf(" • %.0f%% confidence", gen.Confidence*100))
	}

	content := title + "\n\n" + cmd + explText + "\n\n" + riskText
	fmt.Println(commandStyle.Render(content))
}

func ShowBlocked(gen *llm.Generated, result safety.SafetyResult) {
	ensureNewlineAfterPrompt()
	refreshWidth()

	title := errorTitleStyle.Render(blockedIcon + " Command Blocked")
	cmd := ""
	if gen.Command != "" {
		cmd = "\n\n" + lipgloss.NewStyle().Foreground(subtleColor).Render(gen.Command)
	}

	reasons := ""
	for _, reason := range result.Reasons {
		reasons += "\n" + errorIcon + " " + reason
	}

	footer := "\n\n" + lipgloss.NewStyle().Foreground(subtleColor).Italic(true).Render("Use --force to override (not recommended)")

	content := title + cmd + reasons + footer
	fmt.Println(blockedStyle.Render(content))
}

func Confirm(risk safety.RiskLevel) bool {
	// Use fixed-width prefix for alignment: icon or placeholder
	switch risk {
	case safety.RiskHigh:
		red.Print(baseIndent + "⚠ Execute? [y/N]: ")
	default:
		fmt.Print(baseIndent + "  Execute? [Y/n]: ")
	}

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" {
		if risk == safety.RiskHigh {
			return false
		}
		return true
	}
	return input == "y" || input == "yes"
}

var shellOps = map[string]bool{
	"|": true, "||": true, "&&": true, ";": true,
	">": true, ">>": true, "<": true,
}

func tokenizeShellCommand(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	var out []string
	var buf strings.Builder

	inSingle := false
	inDouble := false
	escape := false

	flush := func() {
		if buf.Len() > 0 {
			out = append(out, buf.String())
			buf.Reset()
		}
	}

	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]

		if escape {
			buf.WriteRune(ch)
			escape = false
			continue
		}
		if ch == '\\' && !inSingle {
			escape = true
			continue
		}
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			buf.WriteRune(ch)
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			buf.WriteRune(ch)
			continue
		}

		if !inSingle && !inDouble {
			if ch == ' ' || ch == '\t' || ch == '\n' {
				flush()
				continue
			}

			if i+1 < len(runes) {
				two := string(runes[i : i+2])
				if two == "&&" || two == "||" || two == ">>" {
					flush()
					out = append(out, two)
					i++
					continue
				}
			}
			one := string(ch)
			if one == "|" || one == ";" || one == "<" || one == ">" {
				flush()
				out = append(out, one)
				continue
			}
		}

		buf.WriteRune(ch)
	}

	flush()
	return out
}

type tokenKind int

const (
	kindCommand tokenKind = iota
	kindFlag
	kindArg
	kindOp
)

func classifyTokens(tokens []string) []tokenKind {
	kinds := make([]tokenKind, len(tokens))
	expectCommand := true

	for i, t := range tokens {
		if shellOps[t] {
			kinds[i] = kindOp
			switch t {
			case "|", "||", "&&", ";":
				expectCommand = true
			default:
				expectCommand = false
			}
			continue
		}

		if expectCommand {
			kinds[i] = kindCommand
			expectCommand = false
			continue
		}

		if strings.HasPrefix(t, "-") {
			kinds[i] = kindFlag
		} else {
			kinds[i] = kindArg
		}
	}
	return kinds
}

func ShowLearnMode(gen *llm.Generated) {
	refreshWidth()

	if gen == nil || strings.TrimSpace(gen.Command) == "" {
		return
	}

	cmdTokStyle := lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
	flagTokStyle := lipgloss.NewStyle().Foreground(warningColor)
	argTokStyle := lipgloss.NewStyle().Foreground(successColor)
	opTokStyle := lipgloss.NewStyle().Foreground(accentColor).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(subtleColor)
	sectionTitle := lipgloss.NewStyle().Bold(true).Foreground(infoColor)

	tokens := tokenizeShellCommand(gen.Command)
	kinds := classifyTokens(tokens)

	// Build colorized command preview
	var previewParts []string
	for i, t := range tokens {
		var st lipgloss.Style
		switch kinds[i] {
		case kindCommand:
			st = cmdTokStyle
		case kindFlag:
			st = flagTokStyle
		case kindArg:
			st = argTokStyle
		case kindOp:
			st = opTokStyle
		}
		previewParts = append(previewParts, st.Render(t))
	}
	preview := strings.Join(previewParts, " ")

	// Build breakdown rows
	maxRows := 12
	rows := []string{}
	longCount := 0

	rowCount := len(tokens)
	if rowCount > maxRows {
		longCount = rowCount - maxRows
		rowCount = maxRows
	}

	for i := 0; i < rowCount; i++ {
		t := tokens[i]
		var kindLabel string
		var tok lipgloss.Style
		switch kinds[i] {
		case kindCommand:
			kindLabel = "command"
			tok = cmdTokStyle
		case kindFlag:
			kindLabel = "flag"
			tok = flagTokStyle
		case kindArg:
			kindLabel = "argument"
			tok = argTokStyle
		case kindOp:
			kindLabel = "operator"
			tok = opTokStyle
		}
		rows = append(rows, fmt.Sprintf("  %s  %s", tok.Render(t), labelStyle.Render("← "+kindLabel)))
	}
	if longCount > 0 {
		rows = append(rows, labelStyle.Render(fmt.Sprintf("  … +%d more", longCount)))
	}

	// Explanation section
	var expl string
	if strings.TrimSpace(gen.Explanation) != "" {
		expl = "\n\n" + sectionTitle.Render("💡 Explanation") + "\n" +
			lipgloss.NewStyle().Foreground(subtleColor).Render(gen.Explanation)
	}

	// Alternatives section
	alts := ""
	if len(gen.Alternatives) > 0 {
		var b strings.Builder
		b.WriteString("\n\n")
		b.WriteString(sectionTitle.Render("🔄 Alternatives"))
		b.WriteString("\n")
		for i, a := range gen.Alternatives {
			if i >= 3 {
				b.WriteString(labelStyle.Render(fmt.Sprintf("  … +%d more", len(gen.Alternatives)-i)))
				break
			}
			b.WriteString(fmt.Sprintf("  • %s\n", cmdTokStyle.Render(a.Command)))
			if a.Explanation != "" {
				b.WriteString(fmt.Sprintf("    %s\n", labelStyle.Render(a.Explanation)))
			}
		}
		alts = strings.TrimRight(b.String(), "\n")
	}

	// Compose box content
	title := sectionTitle.Render("📚 Learn Mode")
	content := title +
		"\n\n" + preview +
		"\n\n" + sectionTitle.Render("Parts") + "\n" + strings.Join(rows, "\n") +
		expl +
		alts

	// Use info style box with reduced top margin (no gap between command box)
	learnStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(infoColor).
		Padding(1, 2).
		Margin(0, 2, 1, 2). // top=0, right=2, bottom=1, left=2
		Width(boxWidth)

	fmt.Println(learnStyle.Render(content))
}

func ShowError(err error) {
	ensureNewlineAfterPrompt()
	refreshWidth()
	content := errorTitleStyle.Render(errorIcon+" Error") + "\n\n" + err.Error()
	fmt.Println(errorStyle.Render(content))
}

func ShowRetrying(attempt int) {
	fmt.Println()
	fmt.Println(lipgloss.NewStyle().Foreground(warningColor).Bold(true).Render(
		fmt.Sprintf("  ⟳ Auto-fixing... (attempt %d) [Ctrl+C to stop]", attempt)))
	fmt.Println()
}

func ShowSuccess() {
	fmt.Println(lipgloss.NewStyle().Foreground(successColor).Bold(true).Render("  " + successIcon + " Done"))
}

func ShowOutput(output string) {
	ShowOutputWithCode(output, 0)
}

func ShowOutputWithCode(output string, exitCode int) {
	// Strip ANSI escape codes and control characters
	raw := stripControlChars(output)
	
	// Use separate check string so we don't destroy meaningful indentation
	check := strings.TrimSpace(raw)
	
	// Preserve leading spaces (e.g., wc column alignment); just drop trailing newlines
	cleaned := strings.TrimRight(raw, "\n\r")
	
	// If no output and command succeeded, skip
	if check == "" && exitCode == 0 {
		return
	}
	
	// Show placeholder for empty output on failure
	if check == "" {
		cleaned = lipgloss.NewStyle().Foreground(subtleColor).Italic(true).Render("(no output)")
	}
	
	refreshWidth()
	
	// Build title
	title := lipgloss.NewStyle().Bold(true).Foreground(subtleColor).Render("Output")
	content := title + "\n\n" + cleaned
	
	// Choose style based on exit code
	style := outputStyle
	if exitCode != 0 {
		style = outputStyle.Copy().BorderForeground(errorColor)
		
		// Add footer with exit code inside the box
		separatorWidth := boxWidth - 8 // Account for padding and border
		if separatorWidth < 10 {
			separatorWidth = 10
		}
		separator := lipgloss.NewStyle().Foreground(subtleColor).Render(strings.Repeat("─", separatorWidth))
		
		exitCodeText := lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true).
			Render(fmt.Sprintf("%s Exit code: %d", errorIcon, exitCode))
		
		content += "\n\n" + separator + "\n" + exitCodeText
	} else if exitCode == 0 {
		// Show success indicator for successful commands
		separatorWidth := boxWidth - 8
		if separatorWidth < 10 {
			separatorWidth = 10
		}
		separator := lipgloss.NewStyle().Foreground(subtleColor).Render(strings.Repeat("─", separatorWidth))
		
		successText := lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true).
			Render(fmt.Sprintf("%s Success", successIcon))
		
		content += "\n\n" + separator + "\n" + successText
	}
	
	fmt.Println(style.Render(content))
}

func stripControlChars(s string) string {
	// Remove ANSI escape sequences
	s = ansiRegex.ReplaceAllString(s, "")
	
	// Remove any remaining escape sequences that might have been missed
	// This catches sequences like \x1b[3J, \x1b[H, etc.
	escapeSeqRegex := regexp.MustCompile(`\x1b\[[^\x1b]*`)
	s = escapeSeqRegex.ReplaceAllString(s, "")
	
	// Remove standalone escape character followed by brackets
	s = regexp.MustCompile(`\[[\d;]*[A-Za-z]`).ReplaceAllString(s, "")
	
	// Remove other control characters (except newline and tab)
	var result strings.Builder
	for _, r := range s {
		if r == '\n' || r == '\t' || r >= 32 {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func ShowExitCode(code int) {
	if code != 0 {
		fmt.Println(lipgloss.NewStyle().Foreground(warningColor).Render(fmt.Sprintf("  Exit code: %d", code)))
	}
}

func ShowWelcome() {
	fmt.Println()
	animateLogo()
	fmt.Println()
	
	// Create welcome info box with white border
	refreshWidth()
	
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(textColor).
		Render("Natural Shell v2.0 — AI Terminal Assistant")
	
	hint1 := lipgloss.NewStyle().Foreground(primaryColor).Render("❯ ") +
		lipgloss.NewStyle().Foreground(subtleColor).Render("Type naturally: \"find large files\"")
	
	hint2 := lipgloss.NewStyle().Foreground(infoColor).Render("ℹ ") +
		lipgloss.NewStyle().Foreground(subtleColor).Render("Commands: :help, :diag, :history")
	
	hint3 := lipgloss.NewStyle().Foreground(accentColor).Render("⚡ ") +
		lipgloss.NewStyle().Foreground(subtleColor).Render("Type 'exit' to quit")
	
	content := title + "\n\n" + hint1 + "\n" + hint2 + "\n" + hint3
	
	welcomeBox := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(textColor).
		Padding(1, 2).
		Margin(0, 2).
		Width(boxWidth).
		Render(content)
	
	fmt.Println(welcomeBox)
	fmt.Println()
}

// ShowDryRun displays the command that would be executed in dry-run mode
func ShowDryRun(command string) {
	ensureNewlineAfterPrompt()
	refreshWidth()
	
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(warningColor).
		Render("🔍 DRY RUN")
	
	cmdText := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Render(command)
	
	notice := lipgloss.NewStyle().
		Foreground(subtleColor).
		Italic(true).
		Render("Run without --dry-run to execute")
	
	content := title + "\n\n" + "The following command would be executed:\n\n" + cmdText + "\n\n" + notice
	
	dryRunStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(warningColor).
		Padding(1, 2).
		Margin(1, 2).
		Width(boxWidth)
	
	fmt.Println(dryRunStyle.Render(content))
}

// AskConfirmation prompts the user to confirm command execution
func AskConfirmation(command string) bool {
	ensureNewlineAfterPrompt()
	refreshWidth()
	
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(infoColor).
		Render("⚡ Confirm Execution")
	
	cmdText := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Render(command)
	
	content := title + "\n\n" + cmdText
	
	confirmStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(infoColor).
		Padding(1, 2).
		Margin(1, 2).
		Width(boxWidth)
	
	fmt.Println(confirmStyle.Render(content))
	fmt.Println()
	
	// Prompt for confirmation
	fmt.Print(baseIndent + lipgloss.NewStyle().Foreground(infoColor).Render("Proceed? [y/N]: "))
	
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	
	return input == "y" || input == "yes"
}

// ShowRiskLevel displays the risk level of a command
func ShowRiskLevel(risk safety.RiskLevel) {
	ensureNewlineAfterPrompt()
	
	var icon string
	var riskText string
	var style lipgloss.Style
	
	switch risk {
	case safety.RiskLow:
		icon = successIcon
		riskText = "Low Risk"
		style = lipgloss.NewStyle().Foreground(successColor).Bold(true)
	case safety.RiskMedium:
		icon = warningIcon
		riskText = "Medium Risk"
		style = lipgloss.NewStyle().Foreground(warningColor).Bold(true)
	case safety.RiskHigh:
		icon = errorIcon
		riskText = "High Risk"
		style = lipgloss.NewStyle().Foreground(errorColor).Bold(true)
	case safety.RiskBlocked:
		icon = blockedIcon
		riskText = "Blocked"
		style = lipgloss.NewStyle().Foreground(errorColor).Bold(true)
	default:
		return
	}
	
	fmt.Println(baseIndent + style.Render(fmt.Sprintf("%s %s", icon, riskText)))
}

// ShowInfo displays an informational message
func ShowInfo(message string) {
	ensureNewlineAfterPrompt()
	fmt.Println(baseIndent + lipgloss.NewStyle().Foreground(infoColor).Render("ℹ " + message))
}

// ShowResponse displays the LLM response
func ShowResponse(response *llm.Generated) {
	if response.Message != "" && response.Command == "" {
		ShowAnswer(response.Message)
	}
}

// StartThinking starts the thinking animation and returns a stop function
func StartThinking() func() {
	ShowTranslating()
	return ClearTranslating
}

func animateLogo() {
	logo := []string{
		"  ███╗   ██╗███████╗██╗  ██╗",
		"  ████╗  ██║██╔════╝██║  ██║",
		"  ██╔██╗ ██║███████╗███████║",
		"  ██║╚██╗██║╚════██║██╔══██║",
		"  ██║ ╚████║███████║██║  ██║",
		"  ╚═╝  ╚═══╝╚══════╝╚═╝  ╚═╝",
	}

	colors := []*color.Color{cyan, magenta, blue}

	// Hide cursor during animation
	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h")

	// Animate each line appearing with a sweep effect
	for i, line := range logo {
		c := colors[i%len(colors)]
		runes := []rune(line)
		for j := 0; j <= len(runes); j++ {
			fmt.Print("\r")
			c.Print(string(runes[:j]))
			time.Sleep(8 * time.Millisecond)
		}
		fmt.Println()
	}

	// Color wave animation
	time.Sleep(100 * time.Millisecond)
	for wave := 0; wave < 2; wave++ {
		// Move cursor up 6 lines
		fmt.Print("\033[6A")
		for i, line := range logo {
			colorIdx := (i + wave) % len(colors)
			colors[colorIdx].Println(line)
			time.Sleep(30 * time.Millisecond)
		}
	}
}

func ShowPrompt() {
	fmt.Println() // Add spacing before prompt
	fmt.Print(promptStyle.Render("  nsh "))
	fmt.Print(promptArrowStyle.Render("❯ "))
	promptActive = true
}

// ============ Legacy Box Drawing (kept for compatibility) ============

func printBox(title, content string, borderColor *color.Color) {
	lines := wrapTextWidth(content, legacyBoxWidth-6)
	printBoxWithContent(title, lines, borderColor, white)
}

func printBoxWithContent(title string, lines []string, borderColor, textColor *color.Color) {
	innerWidth := legacyBoxWidth - 4 // Account for "│ " and " │"

	// Pad lines to ensure minimum height
	for len(lines) < legacyBoxMinHeight {
		lines = append(lines, "")
	}

	// Top border: ╭─ Title ─────╮
	borderColor.Print("  ╭─ ")
	borderColor.Print(title)
	borderColor.Print(" ")
	topPadding := legacyBoxWidth - visibleLen(title) - 7
	for i := 0; i < topPadding; i++ {
		borderColor.Print("─")
	}
	borderColor.Println("╮")

	// Content lines
	for i, line := range lines {
		borderColor.Print("  │ ")

		// Clean line for display (replace special chars that cause width issues)
		cleanLine := normalizeText(line)

		// Truncate if needed
		if visibleLen(cleanLine) > innerWidth {
			cleanLine = safeTruncate(cleanLine, innerWidth)
		}

		// Print the line
		if i == len(lines)-1 && isRiskLine(line) {
			textColor.Print(cleanLine)
		} else {
			fmt.Print(cleanLine)
		}

		// Calculate and add padding
		padding := innerWidth - visibleLen(cleanLine)
		if padding > 0 {
			fmt.Print(strings.Repeat(" ", padding))
		}

		borderColor.Println(" │")
	}

	// Bottom border: ╰─────────────╯
	borderColor.Print("  ╰")
	for i := 0; i < legacyBoxWidth-4; i++ {
		borderColor.Print("─")
	}
	borderColor.Println("╯")
}

// ============ Text Utilities ============

// visibleLen returns the visible width of a string
func visibleLen(s string) int {
	// Remove ANSI codes
	clean := ansiRegex.ReplaceAllString(s, "")
	return runewidth.StringWidth(clean)
}

// normalizeText replaces problematic Unicode characters
func normalizeText(s string) string {
	// Replace non-breaking hyphens and other special chars with regular ones
	s = strings.ReplaceAll(s, "‑", "-")  // non-breaking hyphen
	s = strings.ReplaceAll(s, "–", "-")  // en dash
	s = strings.ReplaceAll(s, "—", "-")  // em dash
	s = strings.ReplaceAll(s, "'", "'")  // curly quote
	s = strings.ReplaceAll(s, "'", "'")  // curly quote
	s = strings.ReplaceAll(s, "\u201c", "\"") // left curly quote
	s = strings.ReplaceAll(s, "\u201d", "\"") // right curly quote
	return s
}

// safeTruncate truncates a string to maxWidth visible characters
func safeTruncate(s string, maxWidth int) string {
	s = normalizeText(s)
	if visibleLen(s) <= maxWidth {
		return s
	}

	result := ""
	width := 0
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if width+rw > maxWidth-3 {
			break
		}
		result += string(r)
		width += rw
	}
	return result + "..."
}

// wrapTextWidth wraps text to fit within maxWidth
func wrapTextWidth(text string, maxWidth int) []string {
	text = normalizeText(text)
	var result []string

	paragraphs := strings.Split(text, "\n")
	for _, para := range paragraphs {
		if strings.TrimSpace(para) == "" {
			result = append(result, "")
			continue
		}

		words := strings.Fields(para)
		currentLine := ""
		currentWidth := 0

		for _, word := range words {
			wordWidth := runewidth.StringWidth(word)

			if currentWidth+wordWidth+1 > maxWidth && currentLine != "" {
				result = append(result, currentLine)
				currentLine = word
				currentWidth = wordWidth
			} else {
				if currentLine != "" {
					currentLine += " "
					currentWidth++
				}
				currentLine += word
				currentWidth += wordWidth
			}
		}

		if currentLine != "" {
			result = append(result, currentLine)
		}
	}

	return result
}

func isRiskLine(line string) bool {
	return strings.HasPrefix(line, "✓") ||
		strings.HasPrefix(line, "⚠") ||
		strings.HasPrefix(line, "✗") ||
		strings.HasPrefix(line, "⛔")
}

func getRiskColor(risk safety.RiskLevel) *color.Color {
	switch risk {
	case safety.RiskLow:
		return green
	case safety.RiskMedium:
		return yellow
	case safety.RiskHigh, safety.RiskBlocked:
		return red
	default:
		return dim
	}
}
