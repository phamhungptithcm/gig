package gitlab

import (
	"context"
	"strings"
	"testing"
)

func TestSessionListRepositoriesFiltersNamespaceAndArchived(t *testing.T) {
	t.Parallel()

	session := NewSession(nil, nil, nil)
	session.run = fakeGitLabRunner(map[string]string{
		"projects?membership=true&simple=true&order_by=last_activity_at&sort=desc&per_page=100&page=1": `[
			{"name":"payments","path_with_namespace":"acme/platform/payments","archived":false},
			{"name":"old-ui","path_with_namespace":"acme/platform/old-ui","archived":true},
			{"name":"ops","path_with_namespace":"other/ops","archived":false}
		]`,
		"projects?membership=true&simple=true&order_by=last_activity_at&sort=desc&per_page=100&page=2": `[]`,
	})

	repositories, err := session.ListRepositories(context.Background(), "acme/platform")
	if err != nil {
		t.Fatalf("ListRepositories() error = %v", err)
	}
	if len(repositories) != 1 {
		t.Fatalf("len(repositories) = %d, want 1", len(repositories))
	}
	if repositories[0].Root != "gitlab:acme/platform/payments" {
		t.Fatalf("repositories[0].Root = %q, want gitlab:acme/platform/payments", repositories[0].Root)
	}
}

func TestSessionMissingCLIShowsInstallHint(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	err := NewSession(nil, nil, nil).Status(context.Background())
	if err == nil {
		t.Fatal("Status() error = nil, want missing glab error")
	}

	message := err.Error()
	for _, want := range []string{"glab executable not found", "winget install --id GitLab.cli", "brew install glab", "gig login gitlab"} {
		if !strings.Contains(message, want) {
			t.Fatalf("error = %q, want %q", message, want)
		}
	}
}
