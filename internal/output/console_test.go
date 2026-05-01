package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestConsoleWrapsRowsBulletsAndCommandsToColumns(t *testing.T) {
	t.Setenv("COLUMNS", "48")

	var buffer bytes.Buffer
	ui := NewConsole(&buffer)
	if err := ui.Rows(KeyValue{Label: "Provider", Value: "deep release evidence with checks deployments linked issues and releases"}); err != nil {
		t.Fatalf("Rows() error = %v", err)
	}
	if err := ui.Bullets("Run gig verify ABC-123 with the same scope to turn this evidence into a release verdict."); err != nil {
		t.Fatalf("Bullets() error = %v", err)
	}
	if err := ui.Commands("gig snapshot create ABC-123 --project payments --from staging --to main --format json --output snapshot.json"); err != nil {
		t.Fatalf("Commands() error = %v", err)
	}

	for _, line := range strings.Split(strings.TrimRight(buffer.String(), "\n"), "\n") {
		if len(line) > 48 {
			t.Fatalf("line length = %d, want <= 48: %q\nfull output:\n%s", len(line), line, buffer.String())
		}
	}
}

func TestRenderFrontDoorTruncatesBoxesToColumns(t *testing.T) {
	t.Setenv("COLUMNS", "52")

	var buffer bytes.Buffer
	err := RenderFrontDoor(&buffer, FrontDoorState{
		Version:    "dev",
		HeroStatus: "no project selected yet with a deliberately long explanatory status",
		Prompt:     "ask gig > repo payments with extra detail",
		Examples: []string{
			"repo payments with extra detail that should wrap outside the box",
		},
	})
	if err != nil {
		t.Fatalf("RenderFrontDoor() error = %v", err)
	}

	for _, line := range strings.Split(strings.TrimRight(buffer.String(), "\n"), "\n") {
		if len(line) > 52 {
			t.Fatalf("line length = %d, want <= 52: %q\nfull output:\n%s", len(line), line, buffer.String())
		}
	}
}
