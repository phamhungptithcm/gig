package azuredevops

import (
	"context"
	"net/http"
	"testing"
)

func TestSessionListRepositoriesDiscoversProjectsAndRepos(t *testing.T) {
	t.Parallel()

	session := NewSession(nil, nil, nil)
	session.run = fakeAzureRunner(map[string]string{
		"account get-access-token --resource 499b84ac-1321-427f-aa17-267ca6975798 --query accessToken --output tsv": "token-123",
	})
	session.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Path + "?" + request.URL.RawQuery {
		case "/acme/_apis/projects?$top=100&api-version=7.1":
			return jsonResponse(`{"value":[{"name":"Payments"},{"name":"Core"}]}`), nil
		case "/acme/Payments/_apis/git/repositories?api-version=7.1":
			return jsonResponse(`{"value":[{"name":"release-audit"}]}`), nil
		default:
			t.Fatalf("unexpected request %s?%s", request.URL.Path, request.URL.RawQuery)
			return nil, nil
		}
	})}

	repositories, err := session.ListRepositories(context.Background(), "acme", "Payments")
	if err != nil {
		t.Fatalf("ListRepositories() error = %v", err)
	}
	if len(repositories) != 1 {
		t.Fatalf("len(repositories) = %d, want 1", len(repositories))
	}
	if repositories[0].Root != "azure-devops:acme/Payments/release-audit" {
		t.Fatalf("repositories[0].Root = %q, want azure-devops:acme/Payments/release-audit", repositories[0].Root)
	}
}

func TestSessionListRepositoriesRequiresOrganization(t *testing.T) {
	t.Parallel()

	session := NewSession(nil, nil, nil)
	if _, err := session.ListRepositories(context.Background(), "", ""); err == nil {
		t.Fatal("ListRepositories() error = nil, want organization validation error")
	}
}
