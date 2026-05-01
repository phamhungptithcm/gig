package cli

import (
	"fmt"
	"io"
	"strings"

	"gig/internal/output"
	"gig/internal/sourcecontrol"
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
	_ = ui.Lines("  ", lines...)
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

func printSuggestions(w io.Writer, suggestions []output.FrontDoorSuggestion) {
	if len(suggestions) == 0 {
		return
	}
	ui := newHelpPrinter(w)
	_ = ui.Section("Try next")
	rows := make([]output.KeyValue, 0, len(suggestions))
	flushRows := func() {
		if len(rows) == 0 {
			return
		}
		_ = ui.NestedRows(rows...)
		rows = rows[:0]
	}
	for _, suggestion := range suggestions {
		if suggestion.Command != "" {
			rows = printLabeledHelpSuggestion(ui, rows, suggestion.Label, suggestion.Command, flushRows)
		}
		if suggestion.Note != "" {
			rows = printLabeledHelpSuggestion(ui, rows, suggestion.Label, suggestion.Note, flushRows)
		}
	}
	flushRows()
	_ = ui.Blank()
}

func printLabeledHelpSuggestion(ui output.Console, rows []output.KeyValue, label, value string, flushRows func()) []output.KeyValue {
	label = strings.TrimSpace(label)
	value = strings.TrimSpace(value)
	if value == "" {
		return rows
	}
	if label == "" {
		flushRows()
		_ = ui.Commands(value)
		return rows[:0]
	}
	return append(rows, output.KeyValue{Label: label, Value: value})
}

func providerCoverageHelpRows() []helpRow {
	capabilities := sourcecontrol.OrderedProviderCapabilities()
	rows := make([]helpRow, 0, len(capabilities))
	for _, capability := range capabilities {
		rows = append(rows, helpRow{
			Label: capability.Label,
			Value: capability.Summary(),
		})
	}
	return rows
}
