package svn

import (
	"context"
	"strings"
	"testing"
)

func TestSessionMissingCLIShowsInstallHint(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	err := NewSession(nil, nil, nil).EnsureAuthenticated(context.Background())
	if err == nil {
		t.Fatal("EnsureAuthenticated() error = nil, want missing svn error")
	}

	message := err.Error()
	for _, want := range []string{"svn executable not found", "winget install --id Apache.Subversion", "brew install subversion", "gig login svn"} {
		if !strings.Contains(message, want) {
			t.Fatalf("error = %q, want %q", message, want)
		}
	}
}

func TestRedactCommandArgsHidesSVNPassword(t *testing.T) {
	t.Parallel()

	message := redactCommandArgs([]string{
		"--non-interactive",
		"--username", "demo",
		"--password", "super-secret",
		"info",
		"--xml",
		"https://svn.example.com/repos/app",
	})

	if strings.Contains(message, "super-secret") {
		t.Fatalf("redactCommandArgs() = %q, leaked password", message)
	}
	if !strings.Contains(message, "--password <redacted>") {
		t.Fatalf("redactCommandArgs() = %q, want redacted password marker", message)
	}
}

func TestRedactCommandArgsHidesInlineSVNPassword(t *testing.T) {
	t.Parallel()

	message := redactCommandArgs([]string{"--password=super-secret", "info"})
	if strings.Contains(message, "super-secret") {
		t.Fatalf("redactCommandArgs() = %q, leaked inline password", message)
	}
	if !strings.Contains(message, "--password=<redacted>") {
		t.Fatalf("redactCommandArgs() = %q, want redacted inline password marker", message)
	}
}
