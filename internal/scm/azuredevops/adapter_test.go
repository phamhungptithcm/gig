package azuredevops

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"gig/internal/scm"
	"gig/internal/ticket"
)

func TestAdapterSearchCommitsAcrossDetectedBranches(t *testing.T) {
	t.Parallel()

	parser, err := ticket.NewParser(`\b[A-Z][A-Z0-9]+-\d+\b`)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	adapter := NewAdapter(parser)
	adapter.run = fakeAzureRunner(map[string]string{
		"account show": `{}`,
		"account get-access-token --resource 499b84ac-1321-427f-aa17-267ca6975798 --query accessToken --output tsv": "token-123",
	})
	adapter.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Path + "?" + request.URL.RawQuery {
		case "/acme/Payments/_apis/git/repositories/release-audit?api-version=7.1":
			return jsonResponse(`{"defaultBranch":"refs/heads/main"}`), nil
		case "/acme/Payments/_apis/git/repositories/release-audit/refs?filter=heads/&api-version=7.1":
			return jsonResponse(`{"count":4,"value":[{"name":"refs/heads/feature/test"},{"name":"refs/heads/staging"},{"name":"refs/heads/main"},{"name":"refs/heads/develop"}]}`), nil
		case "/acme/Payments/_apis/git/repositories/release-audit/commits?searchCriteria.$top=100&searchCriteria.itemVersion.versionType=branch&searchCriteria.itemVersion.version=main&api-version=7.1":
			return jsonResponse(`{"count":1,"value":[{"commitId":"abc123","comment":"ABC-123 fix payments"}]}`), nil
		case "/acme/Payments/_apis/git/repositories/release-audit/commits?searchCriteria.$top=100&searchCriteria.itemVersion.versionType=branch&searchCriteria.itemVersion.version=staging&api-version=7.1":
			return jsonResponse(`{"count":2,"value":[{"commitId":"abc123","comment":"ABC-123 fix payments"},{"commitId":"zzz999","comment":"OPS-1 maintenance"}]}`), nil
		case "/acme/Payments/_apis/git/repositories/release-audit/commits?searchCriteria.$top=100&searchCriteria.itemVersion.versionType=branch&searchCriteria.itemVersion.version=develop&api-version=7.1":
			return jsonResponse(`{"count":1,"value":[{"commitId":"abc123","comment":"ABC-123 fix payments"}]}`), nil
		default:
			t.Fatalf("unexpected request %s?%s", request.URL.Path, request.URL.RawQuery)
			return nil, nil
		}
	})}

	commits, err := adapter.SearchCommits(context.Background(), "azure-devops:acme/Payments/release-audit", scm.SearchQuery{TicketID: "ABC-123"})
	if err != nil {
		t.Fatalf("SearchCommits() error = %v", err)
	}

	if len(commits) != 1 {
		t.Fatalf("len(commits) = %d, want 1", len(commits))
	}
	if len(commits[0].Branches) != 3 {
		t.Fatalf("branches = %#v, want commit to be seen in develop, staging, and main", commits[0].Branches)
	}
}

func TestAdapterCompareBranchesReturnsMissingCommits(t *testing.T) {
	t.Parallel()

	parser, err := ticket.NewParser(`\b[A-Z][A-Z0-9]+-\d+\b`)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	adapter := NewAdapter(parser)
	adapter.run = fakeAzureRunner(map[string]string{
		"account show": `{}`,
		"account get-access-token --resource 499b84ac-1321-427f-aa17-267ca6975798 --query accessToken --output tsv": "token-123",
	})
	adapter.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Path + "?" + request.URL.RawQuery {
		case "/acme/Payments/_apis/git/repositories/release-audit/refs?filter=heads/staging&api-version=7.1":
			return jsonResponse(`{"count":1,"value":[{"name":"refs/heads/staging"}]}`), nil
		case "/acme/Payments/_apis/git/repositories/release-audit/refs?filter=heads/main&api-version=7.1":
			return jsonResponse(`{"count":1,"value":[{"name":"refs/heads/main"}]}`), nil
		case "/acme/Payments/_apis/git/repositories/release-audit/commits?searchCriteria.$top=100&searchCriteria.itemVersion.versionType=branch&searchCriteria.itemVersion.version=staging&api-version=7.1":
			return jsonResponse(`{"count":1,"value":[{"commitId":"abc123","comment":"ABC-123 fix payments"}]}`), nil
		case "/acme/Payments/_apis/git/repositories/release-audit/commits?searchCriteria.$top=100&searchCriteria.itemVersion.versionType=branch&searchCriteria.itemVersion.version=main&api-version=7.1":
			return jsonResponse(`{"count":0,"value":[]}`), nil
		default:
			t.Fatalf("unexpected request %s?%s", request.URL.Path, request.URL.RawQuery)
			return nil, nil
		}
	})}

	result, err := adapter.CompareBranches(context.Background(), "azure-devops:acme/Payments/release-audit", scm.CompareQuery{
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

func fakeAzureRunner(outputs map[string]string) commandRunner {
	return func(_ context.Context, args ...string) (string, error) {
		key := strings.Join(args, " ")
		output, ok := outputs[key]
		if !ok {
			return "", errors.New("unexpected az invocation: " + key)
		}
		return output, nil
	}
}
