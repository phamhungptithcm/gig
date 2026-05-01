package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestProgressBarKeepsStableWidth(t *testing.T) {
	t.Parallel()

	bar := progressBar(3)
	if !strings.HasPrefix(bar, "[") || !strings.HasSuffix(bar, "]") {
		t.Fatalf("progressBar() = %q, want bracketed bar", bar)
	}
	if len(bar) != progressWidth+2 {
		t.Fatalf("len(progressBar()) = %d, want %d", len(bar), progressWidth+2)
	}
	if got := strings.Count(bar, "="); got != 5 {
		t.Fatalf("progressBar() = %q, want moving segment of 5", bar)
	}
}

func TestProgressTicketLabel(t *testing.T) {
	t.Parallel()

	if got := progressTicketLabel("verify", []string{"ABC-123"}); got != "verify ABC-123" {
		t.Fatalf("progressTicketLabel(single) = %q", got)
	}
	if got := progressTicketLabel("packet", []string{"ABC-123", "XYZ-999"}); got != "packet 2 tickets" {
		t.Fatalf("progressTicketLabel(batch) = %q", got)
	}
}

func TestProgressClearCoversRenderedLine(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	progress := newProgressIndicator(&buf, strings.Repeat("x", 140), true)
	progress.render(0, time.Second)
	rendered := strings.TrimPrefix(buf.String(), "\r")
	progress.clear()

	parts := strings.Split(buf.String(), "\r")
	if len(parts) < 4 {
		t.Fatalf("progress output = %q, want render and clear carriage returns", buf.String())
	}
	cleared := parts[len(parts)-2]
	if len(cleared) < len(rendered) {
		t.Fatalf("clear wrote %d spaces, want at least rendered line length %d", len(cleared), len(rendered))
	}
}

func TestInteractiveInputPathByOS(t *testing.T) {
	t.Parallel()

	if got := interactiveInputPath("windows"); got != "CONIN$" {
		t.Fatalf("interactiveInputPath(windows) = %q, want CONIN$", got)
	}
	if got := interactiveInputPath("darwin"); got != "/dev/tty" {
		t.Fatalf("interactiveInputPath(darwin) = %q, want /dev/tty", got)
	}
	if got := interactiveInputPath("linux"); got != "/dev/tty" {
		t.Fatalf("interactiveInputPath(linux) = %q, want /dev/tty", got)
	}
}

func TestCommandPromptDisabledWithoutInputOrTerminalOutput(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	app, err := NewAppWithIO(strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("NewAppWithIO() error = %v", err)
	}
	if app.commandPromptEnabled() {
		t.Fatal("commandPromptEnabled() = true, want false without input or terminal output")
	}
}
