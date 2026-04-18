package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

var errPickerCancelled = errors.New("selection cancelled")

const terminalPickerViewport = 8

type pickerEventKind int

const (
	pickerEventUnknown pickerEventKind = iota
	pickerEventUp
	pickerEventDown
	pickerEventSelect
	pickerEventBackspace
	pickerEventCancel
	pickerEventText
)

type pickerEvent struct {
	Kind pickerEventKind
	Text string
}

type pickerModel struct {
	items    []pickerItem
	filtered []pickerItem
	query    string
	selected int
}

func newPickerModel(items []pickerItem) *pickerModel {
	model := &pickerModel{items: append([]pickerItem(nil), items...)}
	model.applyFilter("")
	return model
}

func (m *pickerModel) applyFilter(query string) {
	m.query = query
	m.filtered = filterPickerItems(m.items, query)
	switch {
	case len(m.filtered) == 0:
		m.selected = 0
	case m.selected >= len(m.filtered):
		m.selected = len(m.filtered) - 1
	case m.selected < 0:
		m.selected = 0
	}
}

func (m *pickerModel) appendFilter(text string) {
	m.applyFilter(m.query + text)
}

func (m *pickerModel) backspace() {
	if m.query == "" {
		return
	}
	m.applyFilter(m.query[:len(m.query)-1])
}

func (m *pickerModel) move(delta int) {
	if len(m.filtered) == 0 {
		return
	}
	m.selected += delta
	if m.selected < 0 {
		m.selected = 0
	}
	if m.selected >= len(m.filtered) {
		m.selected = len(m.filtered) - 1
	}
}

func (m *pickerModel) current() (pickerItem, bool) {
	if len(m.filtered) == 0 {
		return pickerItem{}, false
	}
	return m.filtered[m.selected], true
}

func (m *pickerModel) visibleRange(limit int) (int, int) {
	if len(m.filtered) <= limit {
		return 0, len(m.filtered)
	}
	start := m.selected - limit/2
	if start < 0 {
		start = 0
	}
	end := start + limit
	if end > len(m.filtered) {
		end = len(m.filtered)
		start = end - limit
	}
	return start, end
}

func (a *App) terminalPickerEnabled() bool {
	input, inputOK := a.stdin.(*os.File)
	output, outputOK := a.stdout.(*os.File)
	if !inputOK || !outputOK {
		return false
	}
	return term.IsTerminal(int(input.Fd())) && term.IsTerminal(int(output.Fd()))
}

func (a *App) runPicker(reader *bufio.Reader, heading string, items []pickerItem) (pickerItem, error) {
	if len(items) == 0 {
		return pickerItem{}, fmt.Errorf("no choices are available")
	}
	if a.terminalPickerEnabled() {
		return a.runTerminalPicker(heading, items)
	}
	return a.runTextPicker(reader, heading, items)
}

func (a *App) runTextPicker(reader *bufio.Reader, heading string, items []pickerItem) (pickerItem, error) {
	filtered := append([]pickerItem(nil), items...)
	for {
		fmt.Fprintln(a.stdout, heading)
		for index, item := range filtered {
			fmt.Fprintf(a.stdout, "  %d. %s\n", index+1, formatPickerLine(item))
		}
		fmt.Fprint(a.stdout, "Choice or filter text ('/' clears, Enter cancels): ")

		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return pickerItem{}, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			return pickerItem{}, errPickerCancelled
		}
		if line == "/" {
			filtered = append([]pickerItem(nil), items...)
			continue
		}
		index, err := strconv.Atoi(line)
		if err == nil {
			if index < 1 || index > len(filtered) {
				return pickerItem{}, fmt.Errorf("selection %d is out of range", index)
			}
			return filtered[index-1], nil
		}

		next := filterPickerItems(items, line)
		if len(next) == 0 {
			fmt.Fprintf(a.stdout, "No matches for %q.\n", line)
			continue
		}
		if len(next) == 1 {
			return next[0], nil
		}
		filtered = next
	}
}

func (a *App) runTerminalPicker(heading string, items []pickerItem) (pickerItem, error) {
	input := a.stdin.(*os.File)
	state, err := term.MakeRaw(int(input.Fd()))
	if err != nil {
		return pickerItem{}, err
	}
	defer func() {
		_ = term.Restore(int(input.Fd()), state)
	}()

	if _, err := fmt.Fprint(a.stdout, "\033[?25l"); err != nil {
		return pickerItem{}, err
	}
	defer func() {
		_, _ = fmt.Fprint(a.stdout, "\033[?25h")
	}()

	reader := bufio.NewReader(input)
	model := newPickerModel(items)
	renderedLines := 0

	for {
		var renderErr error
		renderedLines, renderErr = renderTerminalPicker(a.stdout, heading, model, renderedLines)
		if renderErr != nil {
			return pickerItem{}, renderErr
		}

		event, err := readPickerEvent(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return pickerItem{}, errPickerCancelled
			}
			return pickerItem{}, err
		}

		switch event.Kind {
		case pickerEventUp:
			model.move(-1)
		case pickerEventDown:
			model.move(1)
		case pickerEventBackspace:
			model.backspace()
		case pickerEventText:
			model.appendFilter(event.Text)
		case pickerEventCancel:
			if _, err := fmt.Fprintln(a.stdout); err != nil {
				return pickerItem{}, err
			}
			return pickerItem{}, errPickerCancelled
		case pickerEventSelect:
			selected, ok := model.current()
			if !ok {
				continue
			}
			if _, err := fmt.Fprintln(a.stdout); err != nil {
				return pickerItem{}, err
			}
			return selected, nil
		}
	}
}

func renderTerminalPicker(w io.Writer, heading string, model *pickerModel, previousLines int) (int, error) {
	if previousLines > 0 {
		if _, err := fmt.Fprintf(w, "\033[%dA\033[J", previousLines); err != nil {
			return 0, err
		}
	}

	lines := 0
	writeLine := func(format string, args ...any) error {
		if _, err := fmt.Fprintf(w, format, args...); err != nil {
			return err
		}
		lines++
		return nil
	}

	if err := writeLine("%s\n", heading); err != nil {
		return 0, err
	}
	if err := writeLine("  Use ↑/↓ to move, Enter/Space to select, type to filter, Backspace to erase, Ctrl+C to cancel.\n"); err != nil {
		return 0, err
	}
	filterLabel := "all results"
	if strings.TrimSpace(model.query) != "" {
		filterLabel = model.query
	}
	if err := writeLine("  Filter: %s\n", filterLabel); err != nil {
		return 0, err
	}

	if len(model.filtered) == 0 {
		if err := writeLine("  No matches yet. Keep typing or press Backspace.\n"); err != nil {
			return 0, err
		}
		return lines, nil
	}

	start, end := model.visibleRange(terminalPickerViewport)
	if start > 0 {
		if err := writeLine("  ... %d more above\n", start); err != nil {
			return 0, err
		}
	}
	for index := start; index < end; index++ {
		prefix := "  "
		if index == model.selected {
			prefix = "› "
		}
		if err := writeLine("%s%s\n", prefix, formatPickerLine(model.filtered[index])); err != nil {
			return 0, err
		}
	}
	if end < len(model.filtered) {
		if err := writeLine("  ... %d more below\n", len(model.filtered)-end); err != nil {
			return 0, err
		}
	}
	return lines, nil
}

func readPickerEvent(reader *bufio.Reader) (pickerEvent, error) {
	b, err := reader.ReadByte()
	if err != nil {
		return pickerEvent{}, err
	}

	switch b {
	case '\r', '\n', ' ':
		return pickerEvent{Kind: pickerEventSelect}, nil
	case 3:
		return pickerEvent{Kind: pickerEventCancel}, nil
	case 8, 127:
		return pickerEvent{Kind: pickerEventBackspace}, nil
	case 'k', 'K':
		return pickerEvent{Kind: pickerEventUp}, nil
	case 'j', 'J':
		return pickerEvent{Kind: pickerEventDown}, nil
	case 0x1b:
		next, err := reader.ReadByte()
		if err != nil {
			return pickerEvent{}, err
		}
		if next != '[' {
			return pickerEvent{Kind: pickerEventUnknown}, nil
		}
		next, err = reader.ReadByte()
		if err != nil {
			return pickerEvent{}, err
		}
		switch next {
		case 'A':
			return pickerEvent{Kind: pickerEventUp}, nil
		case 'B':
			return pickerEvent{Kind: pickerEventDown}, nil
		default:
			return pickerEvent{Kind: pickerEventUnknown}, nil
		}
	default:
		if b >= 32 && b <= 126 {
			return pickerEvent{Kind: pickerEventText, Text: string(b)}, nil
		}
		return pickerEvent{Kind: pickerEventUnknown}, nil
	}
}

func formatPickerLine(item pickerItem) string {
	parts := []string{item.Title}
	if badges := formatPickerBadges(item); badges != "" {
		parts = append(parts, badges)
	}
	line := strings.Join(parts, " ")
	if subtitle := strings.TrimSpace(item.Subtitle); subtitle != "" {
		line = fmt.Sprintf("%s  %s", line, subtitle)
	}
	return line
}

func formatPickerBadges(item pickerItem) string {
	badges := make([]string, 0, 2)
	if item.Current {
		badges = append(badges, "current")
	}
	if item.Recent {
		badges = append(badges, "recent")
	}
	if len(badges) == 0 {
		return ""
	}
	return "[" + strings.Join(badges, ", ") + "]"
}

func filterPickerItems(items []pickerItem, query string) []pickerItem {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return append([]pickerItem(nil), items...)
	}

	filtered := make([]pickerItem, 0, len(items))
	for _, item := range items {
		fields := []string{item.Title, item.Subtitle, item.Value}
		fields = append(fields, item.Keywords...)
		for _, field := range fields {
			if strings.Contains(strings.ToLower(strings.TrimSpace(field)), query) {
				filtered = append(filtered, item)
				break
			}
		}
	}
	return filtered
}
