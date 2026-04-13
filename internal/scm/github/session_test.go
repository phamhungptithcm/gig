package github

import (
	"context"
	"testing"
)

func TestSessionListRepositoriesFiltersOwnerAndArchived(t *testing.T) {
	t.Parallel()

	session := NewSession(nil, nil, nil)
	session.run = fakeGitHubRunner(map[string]string{
		"user/repos?sort=updated&per_page=100&page=1": `[
			{"name":"payments","full_name":"acme/payments","archived":false,"disabled":false,"owner":{"login":"acme"}},
			{"name":"old-service","full_name":"acme/old-service","archived":true,"disabled":false,"owner":{"login":"acme"}},
			{"name":"shared","full_name":"other/shared","archived":false,"disabled":false,"owner":{"login":"other"}}
		]`,
		"user/repos?sort=updated&per_page=100&page=2": `[]`,
	})

	repositories, err := session.ListRepositories(context.Background(), "acme")
	if err != nil {
		t.Fatalf("ListRepositories() error = %v", err)
	}
	if len(repositories) != 1 {
		t.Fatalf("len(repositories) = %d, want 1", len(repositories))
	}
	if repositories[0].Root != "github:acme/payments" {
		t.Fatalf("repositories[0].Root = %q, want github:acme/payments", repositories[0].Root)
	}
}
