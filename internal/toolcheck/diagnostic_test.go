package toolcheck

import (
	"errors"
	"strings"
	"testing"
)

func TestMissingToolDiagnosticIncludesCrossPlatformInstallCommands(t *testing.T) {
	t.Parallel()

	err := MissingTool(GitHubCLI(), errors.New("not found"))
	text := err.Error()
	for _, want := range []string{
		"gh executable not found",
		"brew install gh",
		"winget install --id GitHub.cli",
		"scoop install gh",
		"choco install gh",
		"sudo apt install gh",
		"sudo dnf install gh",
		"sudo pacman -S github-cli",
		"gig login github",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("diagnostic = %q, want %q", text, want)
		}
	}
}

func TestCheckSystemDependenciesMarksRequiredTools(t *testing.T) {
	t.Parallel()

	checks := CheckSystemDependenciesWithLookPath(func(name string) (string, error) {
		if name == "git" {
			return "/usr/bin/git", nil
		}
		return "", errors.New("missing")
	}, map[string]bool{"git": true, "gh": true})

	var gitFound, ghMissing bool
	for _, check := range checks {
		switch check.Name {
		case "git":
			gitFound = check.Required && check.Installed
		case "gh":
			ghMissing = check.Required && !check.Installed
		}
	}
	if !gitFound || !ghMissing {
		t.Fatalf("checks = %#v, want required git found and gh missing", checks)
	}
}
