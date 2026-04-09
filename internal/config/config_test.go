package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"gig/internal/config"
)

func TestLoadForPathUsesDefaultsWhenConfigMissing(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()

	loaded, err := config.LoadForPath(workspace, "")
	if err != nil {
		t.Fatalf("LoadForPath() error = %v", err)
	}

	if loaded.Found {
		t.Fatalf("loaded.Found = true, want false")
	}
	if len(loaded.Config.Environments) != 3 {
		t.Fatalf("len(loaded.Config.Environments) = %d, want 3", len(loaded.Config.Environments))
	}
}

func TestLoadForPathLoadsYAMLAndFindsRepositoryByRelativePath(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	configPath := filepath.Join(workspace, "gig.yaml")
	if err := os.WriteFile(configPath, []byte(`
ticketPattern: '\bABC-\d+\b'
environments:
  - name: dev
    branch: develop
  - name: qa
    branch: release/test
repositories:
  - path: services/a-service
    service: Accounts API
    owner: Backend Team
    kind: app
    notes:
      - Verify login and billing summary
`), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", configPath, err)
	}

	nested := filepath.Join(workspace, "services", "a-service")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", nested, err)
	}

	loaded, err := config.LoadForPath(nested, "")
	if err != nil {
		t.Fatalf("LoadForPath() error = %v", err)
	}

	if !loaded.Found {
		t.Fatalf("loaded.Found = false, want true")
	}
	if loaded.Config.TicketPattern != `\bABC-\d+\b` {
		t.Fatalf("loaded.Config.TicketPattern = %q", loaded.Config.TicketPattern)
	}
	if len(loaded.Config.Environments) != 2 {
		t.Fatalf("len(loaded.Config.Environments) = %d, want 2", len(loaded.Config.Environments))
	}

	repository, ok := loaded.Config.FindRepository(workspace, nested, "a-service")
	if !ok {
		t.Fatalf("FindRepository() ok = false, want true")
	}
	if repository.Service != "Accounts API" {
		t.Fatalf("repository.Service = %q, want %q", repository.Service, "Accounts API")
	}
	if len(repository.Notes) != 1 {
		t.Fatalf("len(repository.Notes) = %d, want 1", len(repository.Notes))
	}
}
