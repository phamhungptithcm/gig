package bitbucket

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSessionLoginPromptsAndStoresCredentials(t *testing.T) {
	authFile := filepath.Join(t.TempDir(), "bitbucket.json")
	t.Setenv("GIG_BITBUCKET_AUTH_FILE", authFile)
	t.Setenv("GIG_BITBUCKET_BASE_URL", "https://bitbucket.example/api/2.0")

	session := NewSession(strings.NewReader("demo@example.com\nsecret-token\n"), nil, nil)
	session.client = statusClient(t, "demo@example.com", "secret-token")
	session.store = fileCredentialStore{path: authFile}

	if err := session.Login(context.Background()); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	content, err := os.ReadFile(authFile)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", authFile, err)
	}
	if !strings.Contains(string(content), "demo@example.com") {
		t.Fatalf("credentials file = %q, want stored email", string(content))
	}
}

func TestSessionListRepositoriesFiltersWorkspace(t *testing.T) {
	t.Parallel()

	session := NewSession(nil, nil, nil)
	session.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/api/2.0/repositories" && request.URL.Path != "/2.0/repositories" && request.URL.Path != "/repositories" {
			t.Fatalf("unexpected request path %s", request.URL.Path)
		}
		return jsonResponse(`{
			"values":[
				{"name":"Payments","full_name":"acme/payments","slug":"payments","workspace":{"slug":"acme"}},
				{"name":"Ops","full_name":"other/ops","slug":"ops","workspace":{"slug":"other"}}
			]
		}`), nil
	})}
	session.store = credentialStoreFunc{
		load: func() (credentials, error) {
			return credentials{Email: "demo@example.com", APIToken: "secret-token"}, nil
		},
	}

	repositories, err := session.ListRepositories(context.Background(), "acme")
	if err != nil {
		t.Fatalf("ListRepositories() error = %v", err)
	}
	if len(repositories) != 1 {
		t.Fatalf("len(repositories) = %d, want 1", len(repositories))
	}
	if repositories[0].Root != "bitbucket:acme/payments" {
		t.Fatalf("repositories[0].Root = %q, want bitbucket:acme/payments", repositories[0].Root)
	}
}

type credentialStoreFunc struct {
	load func() (credentials, error)
	save func(credentials) error
}

func (f credentialStoreFunc) Load() (credentials, error) {
	return f.load()
}

func (f credentialStoreFunc) Save(creds credentials) error {
	if f.save == nil {
		return nil
	}
	return f.save(creds)
}
