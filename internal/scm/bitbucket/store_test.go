package bitbucket

import (
	"net/http"
	"strings"
	"testing"
)

func TestKeychainCredentialStoreLoadParsesStoredJSON(t *testing.T) {
	t.Parallel()

	store := keychainCredentialStore{
		run: func(args ...string) (string, error) {
			if strings.Join(args, " ") != "find-generic-password -a bitbucket-cloud -s com.hunpeolabs.gig.bitbucket -w" {
				t.Fatalf("unexpected security args: %v", args)
			}
			return `{"email":"demo@example.com","apiToken":"secret-token"}`, nil
		},
	}

	creds, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if creds.Email != "demo@example.com" || creds.APIToken != "secret-token" {
		t.Fatalf("credentials = %#v, want stored values", creds)
	}
}

func TestKeychainCredentialStoreSaveWritesJSONPayload(t *testing.T) {
	t.Parallel()

	store := keychainCredentialStore{
		run: func(args ...string) (string, error) {
			got := strings.Join(args, " ")
			if !strings.Contains(got, "add-generic-password -a bitbucket-cloud -s com.hunpeolabs.gig.bitbucket -U -w ") {
				t.Fatalf("unexpected security args: %v", args)
			}
			if !strings.Contains(got, `"email":"demo@example.com"`) || !strings.Contains(got, `"apiToken":"secret-token"`) {
				t.Fatalf("security payload missing credentials: %v", args)
			}
			return "", nil
		},
	}

	if err := store.Save(credentials{Email: "demo@example.com", APIToken: "secret-token"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
}

func statusClient(t *testing.T, email, token string) *http.Client {
	t.Helper()

	return &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/api/2.0/repositories" && request.URL.Path != "/repositories" {
			t.Fatalf("unexpected request path %s", request.URL.Path)
		}
		username, password, ok := request.BasicAuth()
		if !ok {
			t.Fatal("missing basic auth")
		}
		if username != email || password != token {
			t.Fatalf("basic auth = %q/%q, want %s/%s", username, password, email, token)
		}
		return jsonResponse(`{"values":[]}`), nil
	})}
}
