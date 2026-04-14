package cli

import (
	"fmt"
	"io"

	"gig/internal/output"
)

type helpRow struct {
	Label string
	Value string
}

func newHelpPrinter(w io.Writer) output.Console {
	return output.NewConsole(w)
}

func printHelpHeading(w io.Writer, title, subtitle string) {
	ui := newHelpPrinter(w)
	_ = ui.Title(title)
	if subtitle != "" {
		_ = ui.Subtitle(subtitle)
	}
	_ = ui.Blank()
}

func printHelpUsage(w io.Writer, lines ...string) {
	ui := newHelpPrinter(w)
	_ = ui.Section("Usage")
	for _, line := range lines {
		_, _ = fmt.Fprintf(w, "  %s\n", line)
	}
	_ = ui.Blank()
}

func printHelpCommands(w io.Writer, title string, lines ...string) {
	ui := newHelpPrinter(w)
	_ = ui.Section(title)
	_ = ui.Commands(lines...)
	_ = ui.Blank()
}

func printHelpBullets(w io.Writer, title string, lines ...string) {
	ui := newHelpPrinter(w)
	_ = ui.Section(title)
	_ = ui.Bullets(lines...)
	_ = ui.Blank()
}

func printHelpRows(w io.Writer, title string, rows ...helpRow) {
	ui := newHelpPrinter(w)
	_ = ui.Section(title)
	keyValues := make([]output.KeyValue, 0, len(rows))
	for _, row := range rows {
		keyValues = append(keyValues, output.KeyValue{Label: row.Label, Value: row.Value})
	}
	_ = ui.Rows(keyValues...)
	_ = ui.Blank()
}

func printUsageFailure(w io.Writer, command, summary string, examples ...string) {
	ui := newHelpPrinter(w)
	_, _ = fmt.Fprintf(w, "%s failed: %s\n", command, summary)
	if len(examples) == 0 {
		return
	}
	_ = ui.Section("Try next")
	_ = ui.Commands(examples...)
	_ = ui.Blank()
}
