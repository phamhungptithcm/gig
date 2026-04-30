package output

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

type Console struct {
	w      io.Writer
	styled bool
	width  int
}

type KeyValue struct {
	Label string
	Value string
}

func NewConsole(w io.Writer) Console {
	return Console{
		w:      w,
		styled: supportsStyle(w),
		width:  terminalWidth(w),
	}
}

func (c Console) Styled() bool {
	return c.styled
}

func (c Console) Width() int {
	return c.width
}

func (c Console) Blank() error {
	_, err := fmt.Fprintln(c.w)
	return err
}

func (c Console) Title(text string) error {
	_, err := fmt.Fprintln(c.w, c.emphasis(text))
	return err
}

func (c Console) Subtitle(text string) error {
	_, err := fmt.Fprintln(c.w, c.muted(text))
	return err
}

func (c Console) Section(text string) error {
	_, err := fmt.Fprintln(c.w, c.emphasis(text))
	return err
}

func (c Console) Note(text string) error {
	return c.writeWrapped("  ", text, c.muted)
}

func (c Console) Rows(rows ...KeyValue) error {
	return c.writeRows("", rows...)
}

func (c Console) NestedRows(rows ...KeyValue) error {
	return c.writeRows("  ", rows...)
}

func (c Console) Bullets(lines ...string) error {
	return c.writeBullets("  ", lines...)
}

func (c Console) NestedBullets(lines ...string) error {
	return c.writeBullets("    ", lines...)
}

func (c Console) Commands(lines ...string) error {
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if err := c.writeWrapped("  ", line, c.command); err != nil {
			return err
		}
	}
	return nil
}

func (c Console) Lines(indent string, lines ...string) error {
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if err := c.writeWrapped(indent, line, func(value string) string { return value }); err != nil {
			return err
		}
	}
	return nil
}

func (c Console) NestedSection(text string) error {
	_, err := fmt.Fprintf(c.w, "  %s\n", c.emphasis(text))
	return err
}

func (c Console) Emphasis(text string) string {
	return c.emphasis(text)
}

func (c Console) Muted(text string) string {
	return c.muted(text)
}

func (c Console) Command(text string) string {
	return c.command(text)
}

func (c Console) Verdict(text string) string {
	normalized := strings.ToUpper(strings.TrimSpace(text))
	if normalized == "" {
		return ""
	}

	switch strings.ToLower(strings.TrimSpace(text)) {
	case "safe":
		return c.color(normalized, ansiGreen)
	case "warning":
		return c.color(normalized, ansiYellow)
	case "blocked":
		return c.color(normalized, ansiRed)
	default:
		return c.emphasis(normalized)
	}
}

func (c Console) writeRows(indent string, rows ...KeyValue) error {
	width := 0
	filtered := make([]KeyValue, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.Value) == "" {
			continue
		}
		filtered = append(filtered, row)
		if len(row.Label) > width {
			width = len(row.Label)
		}
	}

	for _, row := range filtered {
		label := fmt.Sprintf("%-*s", width, row.Label)
		prefixWidth := len(indent) + width + 2
		valueWidth := c.availableWidth(prefixWidth)
		lines := wrapPlainText(row.Value, valueWidth)
		if len(lines) == 0 {
			continue
		}
		if _, err := fmt.Fprintf(c.w, "%s%s  %s\n", indent, c.muted(label), lines[0]); err != nil {
			return err
		}
		continuationLabel := strings.Repeat(" ", width)
		for _, line := range lines[1:] {
			if _, err := fmt.Fprintf(c.w, "%s%s  %s\n", indent, c.muted(continuationLabel), line); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c Console) writeBullets(indent string, lines ...string) error {
	prefix := "-"
	if c.styled {
		prefix = "•"
	}
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		prefixWidth := len(indent) + len(prefix) + 1
		wrapped := wrapPlainText(line, c.availableWidth(prefixWidth))
		for index, segment := range wrapped {
			marker := prefix
			if index > 0 {
				marker = " "
			}
			if _, err := fmt.Fprintf(c.w, "%s%s %s\n", indent, marker, segment); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c Console) writeWrapped(indent, text string, style func(string) string) error {
	width := c.availableWidth(len(indent))
	lines := wrapPlainText(text, width)
	if len(lines) == 0 {
		return nil
	}
	for _, line := range lines {
		if _, err := fmt.Fprintf(c.w, "%s%s\n", indent, style(line)); err != nil {
			return err
		}
	}
	return nil
}

func (c Console) availableWidth(prefixWidth int) int {
	if c.width <= 0 {
		return 0
	}
	width := c.width - prefixWidth
	if width < 16 {
		return 16
	}
	return width
}

func (c Console) Truncate(text string, width int) string {
	return truncatePlainText(text, width)
}

func (c Console) emphasis(text string) string {
	return c.wrap(text, ansiBold)
}

func (c Console) muted(text string) string {
	return c.wrap(text, ansiDim)
}

func (c Console) command(text string) string {
	return c.wrap(text, ansiCyan)
}

func (c Console) color(text string, code string) string {
	return c.wrap(text, code, ansiBold)
}

func (c Console) wrap(text string, codes ...string) string {
	if !c.styled || strings.TrimSpace(text) == "" {
		return text
	}
	return "\x1b[" + strings.Join(codes, ";") + "m" + text + "\x1b[0m"
}

func supportsStyle(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("TERM")), "dumb") {
		return false
	}
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

func terminalWidth(w io.Writer) int {
	if file, ok := w.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		width, _, err := term.GetSize(int(file.Fd()))
		if err == nil && width >= 40 {
			return width
		}
	}
	if columns := strings.TrimSpace(os.Getenv("COLUMNS")); columns != "" {
		width, err := strconv.Atoi(columns)
		if err == nil && width >= 40 {
			return width
		}
	}
	return 0
}

func wrapPlainText(text string, width int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if width <= 0 || len(text) <= width {
		return []string{text}
	}

	words := strings.Fields(text)
	lines := make([]string, 0, len(words))
	current := ""
	for _, word := range words {
		for len(word) > width {
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
			lines = append(lines, word[:width])
			word = word[width:]
		}
		if word == "" {
			continue
		}
		if current == "" {
			current = word
			continue
		}
		if len(current)+1+len(word) <= width {
			current += " " + word
			continue
		}
		lines = append(lines, current)
		current = word
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func truncatePlainText(text string, width int) string {
	text = strings.TrimRight(text, " ")
	if width <= 0 || len(text) <= width {
		return text
	}
	if width <= 3 {
		return text[:width]
	}
	return strings.TrimRight(text[:width-3], " ") + "..."
}

const (
	ansiBold   = "1"
	ansiDim    = "2"
	ansiRed    = "31"
	ansiGreen  = "32"
	ansiYellow = "33"
	ansiCyan   = "36"
)
