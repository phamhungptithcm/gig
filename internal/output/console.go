package output

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

type Console struct {
	w      io.Writer
	styled bool
}

type KeyValue struct {
	Label string
	Value string
}

func NewConsole(w io.Writer) Console {
	return Console{
		w:      w,
		styled: supportsStyle(w),
	}
}

func (c Console) Styled() bool {
	return c.styled
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
	_, err := fmt.Fprintf(c.w, "  %s\n", c.muted(text))
	return err
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
		if _, err := fmt.Fprintf(c.w, "  %s\n", c.command(line)); err != nil {
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
		if _, err := fmt.Fprintf(c.w, "%s%s  %s\n", indent, c.muted(label), row.Value); err != nil {
			return err
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
		if _, err := fmt.Fprintf(c.w, "%s%s %s\n", indent, prefix, line); err != nil {
			return err
		}
	}
	return nil
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

const (
	ansiBold   = "1"
	ansiDim    = "2"
	ansiRed    = "31"
	ansiGreen  = "32"
	ansiYellow = "33"
	ansiCyan   = "36"
)
