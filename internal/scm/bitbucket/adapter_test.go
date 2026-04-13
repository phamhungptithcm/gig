package bitbucket

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"gig/internal/scm"
	"gig/internal/ticket"
)

func TestAdapterSearchCommitsAcrossProtectedBranches(t *testing.T) {
	t.Parallel()

	parser, err := ticket.NewParser(`\b[A-Z][A-Z0-9]+-\d+\b`)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	adapter := NewAdapter(parser)
	adapter.baseURL = "https://bitbucket.example/api/2.0"
	adapter.credentials = func(context.Context) (credentials, error) {
		return credentials{Email: "demo@example.com", APIToken: "token-123"}, nil
	}
	adapter.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Path + "?" + request.URL.RawQuery {
		case "/api/2.0/repositories/acme/payments?":
			return jsonResponse(`{"mainbranch":{"name":"main"}}`), nil
		case "/api/2.0/repositories/acme/payments/branch-restrictions?pagelen=100&page=1":
			return jsonResponse(`{"values":[{"branch_match_kind":"branching_model","branch_type":"development"},{"branch_match_kind":"branching_model","branch_type":"production"}]}`), nil
		case "/api/2.0/repositories/acme/payments/effective-branching-model?":
			return jsonResponse(`{"development":{"branch":{"name":"staging"}},"production":{"branch":{"name":"main"}}}`), nil
		case "/api/2.0/repositories/acme/payments/commits/staging?pagelen=100&page=1":
			return jsonResponse(`{"values":[{"hash":"abc123456789","message":"ABC-123 fix payments\n\nDepends-On: XYZ-456"},{"hash":"ops987654321","message":"OPS-9 maintenance"}]}`), nil
		case "/api/2.0/repositories/acme/payments/commits/main?pagelen=100&page=1":
			return jsonResponse(`{"values":[{"hash":"abc123456789","message":"ABC-123 fix payments\n\nDepends-On: XYZ-456"}]}`), nil
		default:
			t.Fatalf("unexpected request %s?%s", request.URL.Path, request.URL.RawQuery)
			return nil, nil
		}
	})}

	commits, err := adapter.SearchCommits(context.Background(), "bitbucket:acme/payments", scm.SearchQuery{TicketID: "ABC-123"})
	if err != nil {
		t.Fatalf("SearchCommits() error = %v", err)
	}

	if len(commits) != 1 {
		t.Fatalf("len(commits) = %d, want 1", len(commits))
	}
	if len(commits[0].Branches) != 2 {
		t.Fatalf("branches = %#v, want commit to be seen in staging and main", commits[0].Branches)
	}
}

func TestAdapterCompareBranchesReturnsMissingCommits(t *testing.T) {
	t.Parallel()

	parser, err := ticket.NewParser(`\b[A-Z][A-Z0-9]+-\d+\b`)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	adapter := NewAdapter(parser)
	adapter.baseURL = "https://bitbucket.example/api/2.0"
	adapter.credentials = func(context.Context) (credentials, error) {
		return credentials{Email: "demo@example.com", APIToken: "token-123"}, nil
	}
	adapter.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Path + "?" + request.URL.RawQuery {
		case "/api/2.0/repositories/acme/payments/refs/branches/staging?":
			return jsonResponse(`{"name":"staging"}`), nil
		case "/api/2.0/repositories/acme/payments/refs/branches/main?":
			return jsonResponse(`{"name":"main"}`), nil
		case "/api/2.0/repositories/acme/payments/commits/staging?pagelen=100&page=1":
			return jsonResponse(`{"values":[{"hash":"abc123456789","message":"ABC-123 fix payments"}]}`), nil
		case "/api/2.0/repositories/acme/payments/commits/main?pagelen=100&page=1":
			return jsonResponse(`{"values":[]}`), nil
		default:
			t.Fatalf("unexpected request %s?%s", request.URL.Path, request.URL.RawQuery)
			return nil, nil
		}
	})}

	result, err := adapter.CompareBranches(context.Background(), "bitbucket:acme/payments", scm.CompareQuery{
		TicketID:   "ABC-123",
		FromBranch: "staging",
		ToBranch:   "main",
	})
	if err != nil {
		t.Fatalf("CompareBranches() error = %v", err)
	}

	if len(result.MissingCommits) != 1 {
		t.Fatalf("len(MissingCommits) = %d, want 1", len(result.MissingCommits))
	}
}

func TestAdapterProtectedBranchesFallsBackToBranchSelection(t *testing.T) {
	t.Parallel()

	parser, err := ticket.NewParser(`\b[A-Z][A-Z0-9]+-\d+\b`)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	adapter := NewAdapter(parser)
	adapter.baseURL = "https://bitbucket.example/api/2.0"
	adapter.credentials = func(context.Context) (credentials, error) {
		return credentials{Email: "demo@example.com", APIToken: "token-123"}, nil
	}
	adapter.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Path + "?" + request.URL.RawQuery {
		case "/api/2.0/repositories/acme/payments?":
			return jsonResponse(`{"mainbranch":{"name":"main"}}`), nil
		case "/api/2.0/repositories/acme/payments/branch-restrictions?pagelen=100&page=1":
			return jsonResponse(`{"values":[]}`), nil
		case "/api/2.0/repositories/acme/payments/refs/branches?pagelen=100&page=1":
			return jsonResponse(`{"values":[{"name":"feature/test"},{"name":"develop"},{"name":"main"}]}`), nil
		case "/api/2.0/repositories/acme/payments/effective-branching-model?":
			return jsonResponse(`{"development":{"branch":{"name":"develop"}},"production":{"branch":{"name":"main"}}}`), nil
		default:
			t.Fatalf("unexpected request %s?%s", request.URL.Path, request.URL.RawQuery)
			return nil, nil
		}
	})}

	branches, err := adapter.ProtectedBranches(context.Background(), "bitbucket:acme/payments")
	if err != nil {
		t.Fatalf("ProtectedBranches() error = %v", err)
	}

	if len(branches) != 2 || branches[0] != "develop" || branches[1] != "main" {
		t.Fatalf("branches = %#v, want [develop main]", branches)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
